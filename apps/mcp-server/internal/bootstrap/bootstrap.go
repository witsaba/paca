// Package bootstrap wires the layers of the service together.
//
// It is the only place that knows every layer at once: the domain rules, the
// use cases, the adapters and the infrastructure. Everything else depends on
// a single layer.
package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/http"
	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/inbound/mcpserver"
	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/outbound/database"
	"github.com/witsaba/paca/apps/mcp-server/internal/adapters/outbound/serving"
	"github.com/witsaba/paca/apps/mcp-server/internal/application"
	"github.com/witsaba/paca/apps/mcp-server/internal/platform/config"
	"github.com/witsaba/paca/apps/mcp-server/internal/ports"
	"github.com/witsaba/paca/packages/database/postgres"
	"github.com/witsaba/paca/packages/domain/account"
	"github.com/witsaba/paca/packages/domain/business"
	"github.com/witsaba/paca/packages/domain/health"
	"github.com/witsaba/paca/packages/domain/ingredient"
	"github.com/witsaba/paca/packages/domain/ledger"
	"github.com/witsaba/paca/packages/domain/purchase"
	"github.com/witsaba/paca/packages/domain/recipe"
	"github.com/witsaba/paca/packages/domain/store"
	"github.com/witsaba/paca/packages/domain/user"
	"github.com/witsaba/paca/packages/health/endpoint"
	"github.com/witsaba/paca/packages/health/reporter"
	"github.com/witsaba/paca/packages/platform/clock"
	"github.com/witsaba/paca/packages/platform/server"
)

// Instructions tells the MCP clients what the server is for.
const Instructions = "The PACA MCP server. It speaks the Model Context Protocol " +
	"and it takes care of the people who use PACA: it registers a person and " +
	"starts their onboarding, it moves them to another status, and it " +
	"corrects their data."

// App is the assembled service: the HTTP server that hosts the MCP
// transport, and the health use case that reports on it.
type App struct {
	server           *server.Server
	logger           *slog.Logger
	transportChecker *serving.Checker
	pool             *postgres.Pool
}

// New assembles the service.
//
// The version is the one stamped into the binary at build time. The
// environment overrides it, so a deployment can report the version it wants.
//
// The tracer provider creates the spans of the requests. A service without
// one does not trace.
func New(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	tracerProvider trace.TracerProvider,
	version string,
) (*App, error) {
	serviceVersion := cfg.ServiceVersion
	if serviceVersion == "" {
		serviceVersion = version
	}

	startedAt := time.Now()

	transportChecker := serving.NewChecker()
	checkers := []ports.Checker{transportChecker}

	pool, err := openDatabase(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}
	if pool != nil {
		checkers = append(checkers, database.NewChecker(pool))
	}

	// The health of the service is the shared one: the use case that
	// reports it and the endpoint that answers it are the same in every
	// service of the monorepository.
	healthService, err := reporter.New(
		health.Identity{Name: config.ServiceName, Version: serviceVersion},
		checkers,
		clock.System{},
		startedAt,
		cfg.CheckTimeout,
	)
	if err != nil {
		if pool != nil {
			pool.Close()
		}
		return nil, err
	}

	// The repository of the users is the adapter of the core: the service
	// writes the people through it, and it never writes a row of its own.
	// Without a database there is no repository at all — not a repository
	// that is closed — so the use case tells the agent that calls it that
	// the database is not configured.
	var repository user.Repository
	if pool != nil {
		bound, err := postgres.NewUserRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the users could not be built: %w", err)
		}
		repository = bound
	}

	usersService := application.NewUsersService(repository)

	// The book of a person: the business that owns it, the accounts that
	// hold the money, and the transaction that writes them whole.
	//
	// Without a database there is no book: the use case tells the agent
	// that calls it that the database is not configured.
	var businesses business.Repository
	var accounts account.Repository
	var work store.UnitOfWork
	if pool != nil {
		boundBusinesses, err := postgres.NewBusinessRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the businesses could not be built: %w", err)
		}
		businesses = boundBusinesses

		boundAccounts, err := postgres.NewAccountRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the accounts could not be built: %w", err)
		}
		accounts = boundAccounts

		boundWork, err := postgres.NewUnitOfWork(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the transaction of the storage could not be built: %w", err)
		}
		work = boundWork
	}

	onboardingService := application.NewOnboardingService(repository, businesses, accounts, work)

	// The businesses of a person: the second business of that person, with
	// the book of accounts it starts with, and the list of the businesses
	// they own. Without a database there is no business to open.
	businessService := application.NewBusinessService(businesses, accounts, work)

	// The book of a person: the movements that fill it and the balance that
	// comes out of it. Without a database there is no book.
	var book ledger.Repository
	if pool != nil {
		boundBook, err := postgres.NewLedgerRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the book could not be built: %w", err)
		}
		book = boundBook
	}

	bookService := application.NewBookService(book, clock.System{})

	// The debts of a person and the closing of their month: the debts the
	// book still holds, the payments that close them, and the money of the
	// month. Without a database there is no book and no debt.
	debtService := application.NewDebtService(accounts, book, work, clock.System{})
	summaryService := application.NewSummaryService(accounts, book, clock.System{})

	// The utterance of the owner: the purchase she said, kept as a draft of
	// the business while she decides, and the purchase she confirmed. The
	// confirmation writes the purchase and the money of the payment in one
	// transaction, and it grows the catalog of the names the business
	// buys. Without a database there is no draft and no purchase.
	var drafts purchase.Repository
	var purchases purchase.Recorder
	var ingredients ingredient.Repository
	var layers purchase.LayerRepository
	if pool != nil {
		boundDrafts, err := postgres.NewPurchaseRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the drafts could not be built: %w", err)
		}
		drafts = boundDrafts

		boundPurchases, err := postgres.NewPurchaseRecorder(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the recorder of the purchases could not be built: %w", err)
		}
		purchases = boundPurchases

		boundIngredients, err := postgres.NewIngredientRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the ingredients could not be built: %w", err)
		}
		ingredients = boundIngredients

		boundLayers, err := postgres.NewStockRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the layers could not be built: %w", err)
		}
		layers = boundLayers
	}

	purchaseService := application.NewPurchaseService(drafts, ingredients, clock.System{})
	confirmationService := application.NewConfirmationService(drafts, purchases, accounts, book, ingredients, layers, work, clock.System{})

	// The catalog of the names the business buys: the name the owner reads
	// and the names she uses for it. The tool of the catalog matches the
	// name she said against it, it reads it back with what is known of
	// every ingredient, and it writes what she declares about one of them.
	ingredientService := application.NewIngredientService(ingredients)

	// The recipe of the bakery: the guide the owner confirmed, with its
	// ingredients of the catalog, its steps and its yield. The recipe is
	// written in the book of the business, and without a database there is
	// no recipe to write nor to read.
	var recipes recipe.Repository
	var recipeReader recipe.Reader
	if pool != nil {
		boundRecipes, err := postgres.NewRecipeRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the recipes could not be built: %w", err)
		}
		recipes = boundRecipes
		recipeReader = boundRecipes
	}

	recipeService := application.NewRecipeService(recipes, ingredients)

	// The reference prices: what the market was seen at, where it was read
	// and the day it was read. A reference is never the price of the
	// ingredient: the price the owner pays comes from her own purchases.
	// Without a database there is no reference to write nor to read.
	var references purchase.ReferenceRepository
	if pool != nil {
		boundReferences, err := postgres.NewReferenceRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the reference prices could not be built: %w", err)
		}
		references = boundReferences
	}

	referenceService := application.NewReferenceService(references, ingredients, clock.System{})

	// The stock of the bakery: what the business holds and what it cost.
	// The purchase is the fact and the layer is derived from it, and the
	// valuation is FIFO: the oldest layer is the first one that leaves.
	// Without a database there is no stock to read.
	stockService := application.NewStockService(layers, ingredients)

	// The labor of the owner: the hours she worked, in the three kinds of
	// the labor, and the value of the hour she declared. The labor of a
	// batch is derived from the hours of the steps of the recipe, and it
	// is valued with the value of the hour. Without a database there is
	// no labor to write nor to read.
	var labors recipe.LaborRepository
	var laborReader recipe.LaborReader
	if pool != nil {
		boundLabors, err := postgres.NewLaborRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the labor could not be built: %w", err)
		}
		labors = boundLabors
		laborReader = boundLabors
	}

	laborService := application.NewLaborService(labors, clock.System{})

	// The overhead of the house: the electricity the owner paid, the hours
	// the oven ran and the allocation she declared. The overhead of a
	// batch is derived from the hours of the oven of the steps of the
	// recipe, and it is valued with the rate of the house. Without a
	// database there is no overhead to write nor to read.
	var overheads recipe.OverheadRepository
	var overheadReader recipe.OverheadReader
	if pool != nil {
		boundOverheads, err := postgres.NewOverheadRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the overhead could not be built: %w", err)
		}
		overheads = boundOverheads
		overheadReader = boundOverheads
	}

	overheadService := application.NewOverheadService(overheads, clock.System{})

	// The cost of a batch: what the recipe of the bakery costs right now.
	// The cost is derived from the layers that the recipe consumes, and
	// the valuation is FIFO: the oldest layer is the first one that
	// leaves. Without a database there is no recipe to read and no layer
	// to value.
	batchCostService := application.NewBatchCostService(recipeReader, layers, ingredients, laborReader, overheadReader)

	// The report of the real price: the three layers of the cost of a
	// batch apart - the direct cost, the labor and the overhead - and the
	// price of the owner beside them. The price of the owner is what she
	// declared: it is never inferred. Without a database there is no
	// recipe to read and no layer to value.
	realPriceService := application.NewRealPriceService(recipeReader, layers, ingredients, laborReader, overheadReader)

	// The margin of a product: what the owner charges above the direct
	// cost of the batch, and what the market said about the same
	// ingredients. The reference of the market is read beside the cost,
	// never instead of it: a reference never moves the money of the book.
	// Without a database there is no recipe to read, no layer to value
	// and no reference to compare.
	marginService := application.NewMarginService(batchCostService, references, ingredients, clock.System{})

	// The production of the bakery: what came out of the oven on one day.
	// The production consumes the layers of the stock in the order of the
	// FIFO and makes the units of the yield of the recipe, each one with
	// the day it expires. The production and the layers it consumes are
	// written in one transaction. Without a database there is no
	// production to write.
	var productions recipe.ProductionRepository
	var productionsRead recipe.ProductionReader
	if pool != nil {
		boundProductions, err := postgres.NewProductionRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the productions could not be built: %w", err)
		}
		productions = boundProductions
		productionsRead = boundProductions
	}

	// The corrections of the shelf life: the shelf life of a recipe is
	// approximate until the owner corrects it, and the last correction
	// she declared is the one that counts. Without a database there is no
	// correction to keep and no unit to move.
	var corrections recipe.ShelfLifeRepository
	var shelfLives recipe.ShelfLifeReader
	var units recipe.UnitRepository
	var unitsInTheBook recipe.UnitInTheBookReader

	if pool != nil {
		boundCorrections, err := postgres.NewShelfLifeRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the corrections of the shelf life could not be built: %w", err)
		}
		corrections = boundCorrections
		shelfLives = boundCorrections

		boundUnits, err := postgres.NewUnitRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the units could not be built: %w", err)
		}
		units = boundUnits
		unitsInTheBook = boundUnits
	}

	productionService := application.NewProductionService(
		recipeReader,
		shelfLives,
		layers,
		ingredients,
		productions,
		work,
		clock.System{},
	)

	// The expiry of what the bakery makes: the correction the owner
	// declared of the shelf life, and the units of the business with the
	// day each one expires. The correction and the days it moves are
	// written in one transaction. Without a database there is no unit to
	// read and no correction to keep.
	expiryService := application.NewExpiryService(
		recipeReader,
		corrections,
		units,
		work,
		clock.System{},
	)

	// The waste of the bakery: what she lost, the reason, and the money of
	// what it was worth. The waste takes the units of the product that
	// expire first, and the money of the waste leaves the inventory and
	// lands in the account of the waste: it is never lost from the book.
	// The waste and the movement of its money are written in one
	// transaction. Without a database there is no waste to write.
	var wastes recipe.WasteRepository

	if pool != nil {
		boundWastes, err := postgres.NewWasteRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the wastes could not be built: %w", err)
		}
		wastes = boundWastes
	}

	wasteService := application.NewWasteService(
		productionsRead,
		unitsInTheBook,
		accounts,
		book,
		wastes,
		work,
		clock.System{},
	)

	// The sale of the bakery: what she sold, the money the client paid,
	// and the cost the sale released. The sale takes the units of the
	// product that expire first, and it refuses the sale of a unit that
	// expired. The money of the sale and the cost it releases are written
	// with the sale, in one transaction. Without a database there is no
	// sale to write.
	var sales recipe.SaleRepository

	if pool != nil {
		boundSales, err := postgres.NewSaleRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the sales could not be built: %w", err)
		}
		sales = boundSales
	}

	saleService := application.NewSaleService(
		productionsRead,
		unitsInTheBook,
		accounts,
		book,
		sales,
		work,
		clock.System{},
	)

	// The exit of the book: what the owner consumed and what the bakery
	// gave away. The exit moves no money: it releases the cost of the
	// units that left the book, and the cost leaves the inventory and
	// lands in the account of the expense of the exit. The tax treatment
	// of the exit waits for the professional of the taxes of the owner,
	// and the exit names the question. The exit and the movement of its
	// cost are written in one transaction. Without a database there is no
	// exit to write.
	var exits recipe.ExitRepository

	if pool != nil {
		boundExits, err := postgres.NewExitRepository(pool)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("the repository of the exits could not be built: %w", err)
		}
		exits = boundExits
	}

	exitService := application.NewExitService(
		productionsRead,
		unitsInTheBook,
		accounts,
		book,
		exits,
		work,
		clock.System{},
	)

	mcpServer := mcpserver.New(mcpserver.Options{
		ServiceName:         config.ServiceName,
		ServiceVersion:      serviceVersion,
		Instructions:        Instructions,
		Logger:              logger,
		MaxRequestBodyBytes: cfg.MaxRequestBytes,
		TrustedOrigins:      cfg.TrustedOrigins,
		Users:               usersService,
		Onboarding:          onboardingService,
		OpenBusiness:        businessService,
		ReadBusinesses:      businessService,
		Book:                bookService,
		Debts:               debtService,
		Summary:             summaryService,
		DraftPurchase:       purchaseService,
		ConfirmPurchase:     confirmationService,
		CancelDraft:         purchaseService,
		ReadDrafts:          purchaseService,
		MatchItems:          ingredientService,
		ReadIngredients:     ingredientService,
		CompleteIngredient:  ingredientService,
		RecordRecipe:        recipeService,
		RecordReference:     referenceService,
		ReadReferences:      referenceService,
		ReadStock:           stockService,
		ReadBatchCost:       batchCostService,
		RecordLabor:         laborService,
		RecordOverhead:      overheadService,
		ReadRealPrice:       realPriceService,
		ReadMargin:          marginService,
		RecordProduction:    productionService,
		CorrectShelfLife:    expiryService,
		ReadExpiring:        expiryService,
		RecordWaste:         wasteService,
		RecordSale:          saleService,
		RecordExit:          exitService,
	})

	router := http.NewRouter(http.RouterConfig{
		Health:         healthService,
		MCP:            mcpServer.Handler(),
		Logger:         logger,
		TracerProvider: tracerProvider,
		AllowedHosts:   cfg.AllowedHosts,
	})

	return &App{
		server:           server.New(cfg.Addr(), router, logger, cfg.ShutdownTimeout),
		logger:           logger,
		transportChecker: transportChecker,
		pool:             pool,
	}, nil
}

// openDatabase opens the pool of the database, when the service names one.
//
// The database is optional: a service that does not name it starts without
// one, and its health report says nothing about it. The pool is lazy, so a
// database that is down does not stop the service from starting.
func openDatabase(ctx context.Context, cfg config.Config, logger *slog.Logger) (*postgres.Pool, error) {
	if cfg.Database.URL == "" {
		return nil, nil
	}

	poolCfg := postgres.DefaultConfig()
	poolCfg.URL = cfg.Database.URL
	poolCfg.MaxConns = cfg.Database.MaxConns
	poolCfg.ConnectTimeout = cfg.Database.ConnectTimeout

	pool, err := postgres.Open(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("the pool of the database could not be opened: %w", err)
	}

	logger.Info("the database is configured",
		slog.Int64("max_conns", int64(cfg.Database.MaxConns)),
	)

	return pool, nil
}

// Addr returns the address the server listens on. It is known after Listen.
func (a *App) Addr() string {
	return a.server.Addr()
}

// Listen binds the listener of the server. It is separate from Serve so the
// caller knows the address of a server that binds port zero.
func (a *App) Listen() error {
	return a.server.Listen()
}

// Serve serves the requests until the context is cancelled.
func (a *App) Serve(ctx context.Context) error {
	a.transportChecker.SetServing(true)
	a.logger.Info("the MCP server is serving",
		slog.String("addr", a.Addr()),
		slog.String("mcp_path", http.MCPPath),
		slog.String("health_path", endpoint.Path),
	)
	defer a.transportChecker.SetServing(false)
	defer a.pool.Close()

	return a.server.Serve(ctx)
}

// Run assembles the service and serves it.
func Run(
	ctx context.Context,
	cfg config.Config,
	logger *slog.Logger,
	tracerProvider trace.TracerProvider,
	version string,
) error {
	app, err := New(ctx, cfg, logger, tracerProvider, version)
	if err != nil {
		return err
	}
	return server.Run(ctx, app)
}
