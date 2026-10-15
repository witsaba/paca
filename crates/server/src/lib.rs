//! PACA Server
//!
//! Axum-based REST API server.

pub mod db;
pub mod api;
pub mod error;

use axum::{Router, routing::get};
use tower_http::cors::{Any, CorsLayer};
use tracing_subscriber;

pub async fn run_server(database_url: &str) -> anyhow::Result<()> {
    // Initialize logging
    tracing_subscriber::fmt()
        .with_env_filter(
            tracing_subscriber::EnvFilter::from_default_env()
                .add_directive(tracing::Level::INFO.into()),
        )
        .init();

    // Initialize database
    let pool = db::init_database(database_url).await?;

    // CORS configuration
    let cors = CorsLayer::new()
        .allow_origin(Any)
        .allow_methods(Any)
        .allow_headers(Any);

    // Build routes
    let app = Router::new()
        .route("/health", get(health_check))
        .merge(api::categories::routes())
        .merge(api::transactions::routes())
        .with_state(pool)
        .layer(cors);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await?;
    tracing::info!("Server running on http://0.0.0.0:8080");

    axum::serve(listener, app).await?;

    Ok(())
}

async fn health_check() -> &'static str {
    "OK"
}
