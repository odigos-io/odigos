use axum::{routing::get, Router};
use opentelemetry::trace::TracerProvider;
use opentelemetry_otlp::WithExportConfig;
use opentelemetry_sdk::runtime::Tokio;
use std::net::SocketAddr;
use tower_http::trace::TraceLayer;
use tracing_subscriber::{layer::SubscriberExt, util::SubscriberInitExt};

fn init_tracer() -> Result<(), Box<dyn std::error::Error>> {
    let endpoint = std::env::var("OTEL_EXPORTER_OTLP_ENDPOINT")
        .unwrap_or_else(|_| "http://localhost:4318".to_string());

    let traces_endpoint = format!("{}/v1/traces", endpoint);
    println!("Initializing tracer with endpoint: {}", traces_endpoint);

    let exporter = opentelemetry_otlp::new_exporter()
        .http()
        .with_endpoint(&traces_endpoint);

    let tracer_provider = opentelemetry_otlp::new_pipeline()
        .tracing()
        .with_exporter(exporter)
        .install_batch(Tokio)?;

    let tracer = tracer_provider.tracer("rust-otel-app");

    let telemetry_layer = tracing_opentelemetry::layer().with_tracer(tracer);

    tracing_subscriber::registry()
        .with(tracing_subscriber::EnvFilter::from_default_env()
            .add_directive(tracing::Level::INFO.into()))
        .with(telemetry_layer)
        .with(tracing_subscriber::fmt::layer())
        .init();

    Ok(())
}

#[tracing::instrument]
async fn health() -> &'static str {
    tracing::info!("Health check called");
    "OK"
}

#[tracing::instrument]
async fn hello() -> &'static str {
    tracing::info!("Hello endpoint called");
    do_work().await;
    "Hello from instrumented Rust app!"
}

#[tracing::instrument]
async fn do_work() {
    tracing::info!("Doing some work...");
    tokio::time::sleep(tokio::time::Duration::from_millis(50)).await;
}

#[tokio::main]
async fn main() {
    if let Err(e) = init_tracer() {
        eprintln!("Failed to initialize tracer: {}", e);
    }

    let app = Router::new()
        .route("/health", get(health))
        .route("/", get(hello))
        .layer(TraceLayer::new_for_http());

    let addr = SocketAddr::from(([0, 0, 0, 0], 8080));
    println!("Listening on {}", addr);

    let listener = tokio::net::TcpListener::bind(addr).await.unwrap();
    axum::serve(listener, app).await.unwrap();
}

