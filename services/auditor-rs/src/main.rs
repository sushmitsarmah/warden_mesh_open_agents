mod messages;
mod axl;
mod fetcher;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt::init();
    tracing::info!("auditor: starting");

    let mut stream = axl::subscribe_targets().await;
    while let Some(target) = stream.next().await {
        tracing::info!("received target: {}", target.id);
        // fetch source, run analyzers, publish findings
    }
    Ok(())
}
