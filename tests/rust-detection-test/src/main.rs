use axum::{routing::get, Router};
use std::net::SocketAddr;

async fn health() -> &'static str {
    "OK"
}

async fn hello() -> &'static str {
    "Hello from Rust detection test!"
}

#[tokio::main]
async fn main() {
    let app = Router::new()
        .route("/health", get(health))
        .route("/", get(hello));

    let addr = SocketAddr::from(([0, 0, 0, 0], 8080));
    println!("Listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

