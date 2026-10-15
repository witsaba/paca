//! Shared types for PACA
//!
//! Contains types used by both server and UI crates.

use serde::{Deserialize, Serialize};

/// Transaction type: income or outcome
#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "lowercase")]
pub enum TransactionType {
    Income,
    Outcome,
}

/// Transaction entity
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Transaction {
    pub id: i64,
    pub transaction_type: TransactionType,
    pub amount: f64,
    pub currency: String,
    pub category_id: i64,
    pub description: Option<String>,
    pub label: Option<String>,
    pub date: String,  // ISO 8601 format
    pub created_at: String,
    pub updated_at: String,
}

/// Category entity with hierarchical structure
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Category {
    pub id: i64,
    pub name: String,
    pub parent_id: Option<i64>,
    pub created_at: String,
}

/// Category with full path (for breadcrumb display)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct CategoryWithPath {
    pub category: Category,
    pub path: String, // e.g., "Food > Groceries"
}

/// Currency codes (ISO 4217 subset)
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Currency {
    pub code: String,
    pub symbol: String,
    pub name: String,
}

/// Supported currencies
pub const SUPPORTED_CURRENCIES: &[(&str, &str, &str)] = &[
    ("COP", "$", "Colombian Peso"),
    ("USD", "$", "US Dollar"),
    ("EUR", "€", "Euro"),
    ("GBP", "£", "British Pound"),
];
