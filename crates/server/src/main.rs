//! PACA Server Binary

use std::env;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    let database_url = env::var("DATABASE_URL").unwrap_or_else(|_| "paca.db".to_string());

    server::run_server(&database_url).await?;

    Ok(())
}
