//! Transactions API

use crate::db::Database;
use crate::error::AppError;
use axum::{
    extract::{Path, Query, State},
    routing::{get, post, put, delete},
    Router,
    response::IntoResponse,
    Json,
};
use serde::Deserialize;
use shared::{Transaction, TransactionType};

/// Query parameters for listing transactions
#[derive(Debug, Deserialize)]
pub struct ListTransactionsQuery {
    pub transaction_type: Option<String>,
    pub category_id: Option<i64>,
    pub currency: Option<String>,
    pub date_from: Option<String>,
    pub date_to: Option<String>,
    pub sort_by: Option<String>,
    pub sort_order: Option<String>,
}

/// Create a new transaction
#[derive(Debug, Deserialize)]
pub struct CreateTransaction {
    pub transaction_type: TransactionType,
    pub amount: f64,
    pub currency: String,
    pub category_id: i64,
    pub description: Option<String>,
    pub label: Option<String>,
    pub date: String,
}

/// Update a transaction
#[derive(Debug, Deserialize)]
pub struct UpdateTransaction {
    pub transaction_type: Option<TransactionType>,
    pub amount: Option<f64>,
    pub currency: Option<String>,
    pub category_id: Option<i64>,
    pub description: Option<String>,
    pub label: Option<String>,
    pub date: Option<String>,
}

/// Get routes for transactions API
pub fn routes() -> Router<Database> {
    Router::new()
        .route("/api/transactions", get(list_transactions))
        .route("/api/transactions", post(create_transaction))
        .route("/api/transactions/:id", get(get_transaction))
        .route("/api/transactions/:id", put(update_transaction))
        .route("/api/transactions/:id", delete(delete_transaction))
}

fn row_to_transaction(row: &rusqlite::Row) -> rusqlite::Result<Transaction> {
    Ok(Transaction {
        id: row.get(0)?,
        transaction_type: match row.get::<_, String>(1)?.as_str() {
            "income" => TransactionType::Income,
            _ => TransactionType::Outcome,
        },
        amount: row.get(2)?,
        currency: row.get(3)?,
        category_id: row.get(4)?,
        description: row.get(5)?,
        label: row.get(6)?,
        date: row.get(7)?,
        created_at: row.get(8)?,
        updated_at: row.get(9)?,
    })
}

/// GET /api/transactions
async fn list_transactions(
    State(db): State<Database>,
    Query(params): Query<ListTransactionsQuery>,
) -> Result<impl IntoResponse, AppError> {
    let conn = db.connection();
    let guard = conn.lock();

    let mut sql = String::from(
        "SELECT id, transaction_type, amount, currency, category_id, description, label, date, created_at, updated_at FROM transactions WHERE 1=1"
    );
    let mut query_params: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();

    if let Some(ref tt) = params.transaction_type {
        sql.push_str(" AND transaction_type = ?");
        query_params.push(Box::new(tt.clone()));
    }
    if let Some(cid) = params.category_id {
        sql.push_str(" AND category_id = ?");
        query_params.push(Box::new(cid));
    }
    if let Some(ref cur) = params.currency {
        sql.push_str(" AND currency = ?");
        query_params.push(Box::new(cur.clone()));
    }
    if let Some(ref df) = params.date_from {
        sql.push_str(" AND date >= ?");
        query_params.push(Box::new(df.clone()));
    }
    if let Some(ref dt) = params.date_to {
        sql.push_str(" AND date <= ?");
        query_params.push(Box::new(dt.clone()));
    }

    let sort_by = params.sort_by.as_deref().unwrap_or("date");
    let sort_dir = params.sort_order.as_deref().unwrap_or("desc");
    let order = match sort_by {
        "amount" => "amount",
        "category" => "category_id",
        _ => "date",
    };
    let dir = match sort_dir {
        "asc" => "ASC",
        _ => "DESC",
    };
    sql.push_str(&format!(" ORDER BY {} {}", order, dir));

    let mut stmt = guard.prepare(&sql)?;
    let rows = stmt.query_map(rusqlite::params_from_iter(query_params.iter()), row_to_transaction)?;
    let transactions: Vec<Transaction> = rows.filter_map(|r| r.ok()).collect();

    Ok(Json(transactions))
}

/// GET /api/transactions/:id
async fn get_transaction(
    State(db): State<Database>,
    Path(id): Path<i64>,
) -> Result<impl IntoResponse, AppError> {
    let conn = db.connection();
    let guard = conn.lock();
    let mut stmt = guard.prepare(
        "SELECT id, transaction_type, amount, currency, category_id, description, label, date, created_at, updated_at FROM transactions WHERE id = ?"
    )?;
    let transaction = stmt.query_row([id], row_to_transaction)?;
    drop(stmt);
    drop(guard);

    Ok(Json(transaction))
}

/// POST /api/transactions
async fn create_transaction(
    State(db): State<Database>,
    Json(payload): Json<CreateTransaction>,
) -> Result<impl IntoResponse, AppError> {
    if payload.amount <= 0.0 {
        return Err(AppError::Validation("Amount must be positive".into()));
    }
    if payload.currency.trim().is_empty() {
        return Err(AppError::Validation("Currency cannot be empty".into()));
    }

    let conn = db.connection();
    let guard = conn.lock();

    let cat_exists: i64 = guard.query_row(
        "SELECT COUNT(*) FROM categories WHERE id = ?",
        [payload.category_id],
        |row| row.get(0),
    )?;
    if cat_exists == 0 {
        return Err(AppError::Validation("Category not found".into()));
    }

    guard.execute(
        "INSERT INTO transactions (transaction_type, amount, currency, category_id, description, label, date) VALUES (?, ?, ?, ?, ?, ?, ?)",
        (
            match payload.transaction_type {
                TransactionType::Income => "income",
                TransactionType::Outcome => "outcome",
            },
            payload.amount,
            &payload.currency,
            payload.category_id,
            &payload.description,
            &payload.label,
            &payload.date,
        ),
    )?;

    let id = guard.last_insert_rowid();
    drop(guard);

    // Fetch created transaction
    let conn = db.connection();
    let guard = conn.lock();
    let mut stmt = guard.prepare(
        "SELECT id, transaction_type, amount, currency, category_id, description, label, date, created_at, updated_at FROM transactions WHERE id = ?"
    )?;
    let transaction = stmt.query_row([id], row_to_transaction)?;
    drop(stmt);
    drop(guard);

    Ok(Json(transaction))
}

/// PUT /api/transactions/:id
async fn update_transaction(
    State(db): State<Database>,
    Path(id): Path<i64>,
    Json(payload): Json<UpdateTransaction>,
) -> Result<impl IntoResponse, AppError> {
    let conn = db.connection();
    let guard = conn.lock();

    let mut updates = Vec::new();
    let mut values: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();

    if let Some(tt) = payload.transaction_type {
        updates.push("transaction_type = ?");
        values.push(Box::new(match tt {
            TransactionType::Income => "income".to_string(),
            TransactionType::Outcome => "outcome".to_string(),
        }));
    }
    if let Some(amount) = payload.amount {
        if amount <= 0.0 {
            return Err(AppError::Validation("Amount must be positive".into()));
        }
        updates.push("amount = ?");
        values.push(Box::new(amount));
    }
    if let Some(currency) = payload.currency {
        if currency.trim().is_empty() {
            return Err(AppError::Validation("Currency cannot be empty".into()));
        }
        updates.push("currency = ?");
        values.push(Box::new(currency));
    }
    if let Some(category_id) = payload.category_id {
        let cat_exists: i64 = guard.query_row(
            "SELECT COUNT(*) FROM categories WHERE id = ?",
            [category_id],
            |row| row.get(0),
        )?;
        if cat_exists == 0 {
            return Err(AppError::Validation("Category not found".into()));
        }
        updates.push("category_id = ?");
        values.push(Box::new(category_id));
    }
    if let Some(description) = payload.description {
        updates.push("description = ?");
        values.push(Box::new(description));
    }
    if let Some(label) = payload.label {
        updates.push("label = ?");
        values.push(Box::new(label));
    }
    if let Some(date) = payload.date {
        updates.push("date = ?");
        values.push(Box::new(date));
    }

    updates.push("updated_at = datetime('now')");

    let sql = format!("UPDATE transactions SET {} WHERE id = ?", updates.join(", "));
    values.push(Box::new(id));

    guard.execute(&sql, rusqlite::params_from_iter(values.iter()))?;
    drop(guard);

    // Fetch updated transaction
    let conn = db.connection();
    let guard = conn.lock();
    let mut stmt = guard.prepare(
        "SELECT id, transaction_type, amount, currency, category_id, description, label, date, created_at, updated_at FROM transactions WHERE id = ?"
    )?;
    let transaction = stmt.query_row([id], row_to_transaction)?;
    drop(stmt);
    drop(guard);

    Ok(Json(transaction))
}

/// DELETE /api/transactions/:id
async fn delete_transaction(
    State(db): State<Database>,
    Path(id): Path<i64>,
) -> Result<impl IntoResponse, AppError> {
    let conn = db.connection();
    let guard = conn.lock();
    guard.execute("DELETE FROM transactions WHERE id = ?", [id])?;
    Ok(Json(serde_json::json!({"deleted": true})))
}
