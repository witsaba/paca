//! Categories API

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
use shared::{Category, CategoryWithPath};

/// Query parameters for listing categories
#[derive(Debug, Deserialize)]
pub struct ListCategoriesQuery {
    pub parent_id: Option<i64>,
}

/// Create a new category
#[derive(Debug, Deserialize)]
pub struct CreateCategory {
    pub name: String,
    pub parent_id: Option<i64>,
}

/// Update a category
#[derive(Debug, Deserialize)]
pub struct UpdateCategory {
    pub name: Option<String>,
    pub parent_id: Option<Option<i64>>,
}

/// Get routes for categories API
pub fn routes() -> Router<Database> {
    Router::new()
        .route("/api/categories", get(list_categories))
        .route("/api/categories", post(create_category))
        .route("/api/categories/:id", get(get_category))
        .route("/api/categories/:id", put(update_category))
        .route("/api/categories/:id", delete(delete_category))
}

fn row_to_category(row: &rusqlite::Row) -> rusqlite::Result<Category> {
    Ok(Category {
        id: row.get(0)?,
        name: row.get(1)?,
        parent_id: row.get(2)?,
        created_at: row.get(3)?,
    })
}

/// GET /api/categories
async fn list_categories(
    State(db): State<Database>,
    Query(params): Query<ListCategoriesQuery>,
) -> Result<impl IntoResponse, AppError> {
    let conn = db.connection();
    let guard = conn.lock();

    let categories: Vec<Category> = if let Some(pid) = params.parent_id {
        let mut stmt = guard.prepare("SELECT id, name, parent_id, created_at FROM categories WHERE parent_id = ?")?;
        let rows = stmt.query_map([pid], row_to_category)?;
        let result: Vec<Category> = rows.filter_map(|r| r.ok()).collect();
        result
    } else {
        let mut stmt = guard.prepare("SELECT id, name, parent_id, created_at FROM categories")?;
        let rows = stmt.query_map([], row_to_category)?;
        let result: Vec<Category> = rows.filter_map(|r| r.ok()).collect();
        result
    };

    drop(guard);

    // Build paths for all categories
    let mut result = Vec::new();
    for cat in categories {
        let path = build_category_path(&cat, &db)?;
        result.push(CategoryWithPath { category: cat, path });
    }

    Ok(Json(result))
}

/// Build category path (e.g., "Food > Groceries")
fn build_category_path(category: &Category, db: &Database) -> Result<String, AppError> {
    let mut path_parts = vec![category.name.clone()];
    let mut current = category.clone();

    while let Some(parent_id) = current.parent_id {
        let conn = db.connection();
        let guard = conn.lock();
        let mut stmt = guard.prepare("SELECT id, name, parent_id, created_at FROM categories WHERE id = ?")?;
        let parent: Category = stmt.query_row([parent_id], row_to_category)?;
        drop(stmt);
        drop(guard);

        path_parts.insert(0, parent.name.clone());
        current = parent;
    }

    Ok(path_parts.join(" > "))
}

/// GET /api/categories/:id
async fn get_category(
    State(db): State<Database>,
    Path(id): Path<i64>,
) -> Result<impl IntoResponse, AppError> {
    let conn = db.connection();
    let guard = conn.lock();
    let mut stmt = guard.prepare("SELECT id, name, parent_id, created_at FROM categories WHERE id = ?")?;
    let category: Category = stmt.query_row([id], row_to_category)?;
    drop(stmt);
    drop(guard);

    let path = build_category_path(&category, &db)?;

    Ok(Json(CategoryWithPath { category, path }))
}

/// POST /api/categories
async fn create_category(
    State(db): State<Database>,
    Json(payload): Json<CreateCategory>,
) -> Result<impl IntoResponse, AppError> {
    if payload.name.trim().is_empty() {
        return Err(AppError::Validation("Category name cannot be empty".into()));
    }

    let conn = db.connection();
    let guard = conn.lock();
    guard.execute(
        "INSERT INTO categories (name, parent_id) VALUES (?, ?)",
        (&payload.name, &payload.parent_id),
    )?;
    let id = guard.last_insert_rowid();
    drop(guard);

    // Fetch the created category
    let conn = db.connection();
    let guard = conn.lock();
    let mut stmt = guard.prepare("SELECT id, name, parent_id, created_at FROM categories WHERE id = ?")?;
    let category: Category = stmt.query_row([id], row_to_category)?;
    drop(stmt);
    drop(guard);

    let path = build_category_path(&category, &db)?;
    Ok(Json(CategoryWithPath { category, path }))
}

/// PUT /api/categories/:id
async fn update_category(
    State(db): State<Database>,
    Path(id): Path<i64>,
    Json(payload): Json<UpdateCategory>,
) -> Result<impl IntoResponse, AppError> {
    let conn = db.connection();
    let guard = conn.lock();

    // Build update query dynamically
    let mut updates = Vec::new();
    let mut values: Vec<Box<dyn rusqlite::ToSql>> = Vec::new();

    if let Some(name) = &payload.name {
        if name.trim().is_empty() {
            return Err(AppError::Validation("Category name cannot be empty".into()));
        }
        updates.push("name = ?");
        values.push(Box::new(name.clone()));
    }

    updates.push("parent_id = ?");
    values.push(Box::new(payload.parent_id.flatten()));

    let sql = format!("UPDATE categories SET {} WHERE id = ?", updates.join(", "));
    values.push(Box::new(id));

    guard.execute(&sql, rusqlite::params_from_iter(values.iter()))?;
    drop(guard);

    // Fetch the updated category
    let conn = db.connection();
    let guard = conn.lock();
    let mut stmt = guard.prepare("SELECT id, name, parent_id, created_at FROM categories WHERE id = ?")?;
    let category: Category = stmt.query_row([id], row_to_category)?;
    drop(stmt);
    drop(guard);

    let path = build_category_path(&category, &db)?;
    Ok(Json(CategoryWithPath { category, path }))
}

/// DELETE /api/categories/:id
async fn delete_category(
    State(db): State<Database>,
    Path(id): Path<i64>,
) -> Result<impl IntoResponse, AppError> {
    let conn = db.connection();
    let guard = conn.lock();

    // Check if category has children
    let child_count: i64 = guard.query_row(
        "SELECT COUNT(*) FROM categories WHERE parent_id = ?",
        [id],
        |row| row.get(0),
    )?;

    if child_count > 0 {
        return Err(AppError::Validation(
            "Cannot delete category with children. Delete children first.".into(),
        ));
    }

    // Check if category is used by transactions
    let tx_count: i64 = guard.query_row(
        "SELECT COUNT(*) FROM transactions WHERE category_id = ?",
        [id],
        |row| row.get(0),
    )?;

    if tx_count > 0 {
        return Err(AppError::Validation(
            "Cannot delete category used by transactions.".into(),
        ));
    }

    guard.execute("DELETE FROM categories WHERE id = ?", [id])?;

    Ok(Json(serde_json::json!({"deleted": true})))
}
