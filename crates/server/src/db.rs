//! Database initialization and connection pool

use rusqlite::Connection;
use std::sync::Arc;
use parking_lot::Mutex;

/// Database connection pool (single connection for SQLite)
#[derive(Clone)]
pub struct Database {
    conn: Arc<Mutex<Connection>>,
}

impl Database {
    pub fn new(path: &str) -> Result<Self, rusqlite::Error> {
        let conn = Connection::open(path)?;
        conn.execute_batch("PRAGMA foreign_keys = ON;")?;
        Ok(Self {
            conn: Arc::new(Mutex::new(conn)),
        })
    }

    pub fn connection(&self) -> Arc<Mutex<Connection>> {
        Arc::clone(&self.conn)
    }
}

/// Initialize database with schema
pub async fn init_database(database_url: &str) -> Result<Database, rusqlite::Error> {
    let db = Database::new(database_url)?;

    let conn = db.connection();
    let conn = conn.lock();

    conn.execute_batch(
        r#"
        CREATE TABLE IF NOT EXISTS categories (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            parent_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
            created_at TEXT NOT NULL DEFAULT (datetime('now'))
        );

        CREATE TABLE IF NOT EXISTS transactions (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            transaction_type TEXT NOT NULL CHECK(transaction_type IN ('income', 'outcome')),
            amount REAL NOT NULL,
            currency TEXT NOT NULL,
            category_id INTEGER NOT NULL REFERENCES categories(id),
            description TEXT,
            label TEXT,
            date TEXT NOT NULL,
            created_at TEXT NOT NULL DEFAULT (datetime('now')),
            updated_at TEXT NOT NULL DEFAULT (datetime('now'))
        );

        CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(date);
        CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions(category_id);
        CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(transaction_type);
        "#,
    )?;

    tracing::info!("Database initialized at {}", database_url);

    Ok(db)
}
