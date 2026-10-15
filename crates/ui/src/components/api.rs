//! API client for PACA backend

use serde::{Deserialize, Serialize};
use shared::{Category, CategoryWithPath, Transaction, TransactionType};

const API_BASE: &str = "/api";

#[derive(Debug, Serialize)]
pub struct CreateCategory {
    pub name: String,
    pub parent_id: Option<i64>,
}

#[derive(Debug, Serialize)]
pub struct UpdateCategory {
    pub name: Option<String>,
    pub parent_id: Option<Option<i64>>,
}

#[derive(Debug, Serialize)]
pub struct CreateTransaction {
    pub transaction_type: TransactionType,
    pub amount: f64,
    pub currency: String,
    pub category_id: i64,
    pub description: Option<String>,
    pub label: Option<String>,
    pub date: String,
}

#[derive(Debug, Serialize)]
pub struct UpdateTransaction {
    pub transaction_type: Option<TransactionType>,
    pub amount: Option<f64>,
    pub currency: Option<String>,
    pub category_id: Option<i64>,
    pub description: Option<String>,
    pub label: Option<String>,
    pub date: Option<String>,
}

// Categories API
pub async fn list_categories(parent_id: Option<i64>) -> Result<Vec<CategoryWithPath>, String> {
    let url = match parent_id {
        Some(pid) => format!("{}/categories?parent_id={}", API_BASE, pid),
        None => format!("{}/categories", API_BASE),
    };
    let resp = gloo_net::http::Request::get(&url)
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    resp.json::<Vec<CategoryWithPath>>()
        .await
        .map_err(|e| e.to_string())
}

pub async fn get_category(id: i64) -> Result<CategoryWithPath, String> {
    let url = format!("{}/categories/{}", API_BASE, id);
    let resp = gloo_net::http::Request::get(&url)
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    resp.json::<CategoryWithPath>()
        .await
        .map_err(|e| e.to_string())
}

pub async fn create_category(payload: CreateCategory) -> Result<Category, String> {
    let resp = gloo_net::http::Request::post(&format!("{}/categories", API_BASE))
        .json(&payload)
        .map_err(|e| e.to_string())?
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    resp.json::<Category>()
        .await
        .map_err(|e| e.to_string())
}

pub async fn update_category(id: i64, payload: UpdateCategory) -> Result<Category, String> {
    let resp = gloo_net::http::Request::put(&format!("{}/categories/{}", API_BASE, id))
        .json(&payload)
        .map_err(|e| e.to_string())?
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    resp.json::<Category>()
        .await
        .map_err(|e| e.to_string())
}

pub async fn delete_category(id: i64) -> Result<(), String> {
    let resp = gloo_net::http::Request::delete(&format!("{}/categories/{}", API_BASE, id))
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    Ok(())
}

// Transactions API
#[derive(Debug, Deserialize)]
pub struct TransactionListParams {
    pub transaction_type: Option<String>,
    pub category_id: Option<i64>,
    pub currency: Option<String>,
    pub date_from: Option<String>,
    pub date_to: Option<String>,
    pub sort_by: Option<String>,
    pub sort_order: Option<String>,
}

pub async fn list_transactions(params: Option<TransactionListParams>) -> Result<Vec<Transaction>, String> {
    let url = match params {
        Some(p) => {
            let mut query = vec![];
            if let Some(tt) = p.transaction_type {
                query.push(format!("transaction_type={}", tt));
            }
            if let Some(cid) = p.category_id {
                query.push(format!("category_id={}", cid));
            }
            if let Some(cur) = p.currency {
                query.push(format!("currency={}", cur));
            }
            if let Some(df) = p.date_from {
                query.push(format!("date_from={}", df));
            }
            if let Some(dt) = p.date_to {
                query.push(format!("date_to={}", dt));
            }
            if let Some(sb) = p.sort_by {
                query.push(format!("sort_by={}", sb));
            }
            if let Some(so) = p.sort_order {
                query.push(format!("sort_order={}", so));
            }
            if query.is_empty() {
                format!("{}/transactions", API_BASE)
            } else {
                format!("{}/transactions?{}", API_BASE, query.join("&"))
            }
        }
        None => format!("{}/transactions", API_BASE),
    };
    
    let resp = gloo_net::http::Request::get(&url)
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    resp.json::<Vec<Transaction>>()
        .await
        .map_err(|e| e.to_string())
}

pub async fn create_transaction(payload: CreateTransaction) -> Result<Transaction, String> {
    let resp = gloo_net::http::Request::post(&format!("{}/transactions", API_BASE))
        .json(&payload)
        .map_err(|e| e.to_string())?
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    resp.json::<Transaction>()
        .await
        .map_err(|e| e.to_string())
}

pub async fn update_transaction(id: i64, payload: UpdateTransaction) -> Result<Transaction, String> {
    let resp = gloo_net::http::Request::put(&format!("{}/transactions/{}", API_BASE, id))
        .json(&payload)
        .map_err(|e| e.to_string())?
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    resp.json::<Transaction>()
        .await
        .map_err(|e| e.to_string())
}

pub async fn delete_transaction(id: i64) -> Result<(), String> {
    let resp = gloo_net::http::Request::delete(&format!("{}/transactions/{}", API_BASE, id))
        .send()
        .await
        .map_err(|e| e.to_string())?;
    
    if !resp.ok() {
        return Err(format!("HTTP error: {}", resp.status()));
    }
    
    Ok(())
}
