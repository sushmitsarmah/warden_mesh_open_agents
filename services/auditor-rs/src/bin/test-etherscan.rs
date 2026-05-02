use auditor::fetcher::fetch_source;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    println!("🔍 Testing Etherscan API Integration\n");

    // Test with VulnerableVault contract on Sepolia (if deployed)
    // For now, use a well-known verified contract: USDC on mainnet
    let test_address = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"; // USDC on Ethereum mainnet
    let chain_id = 1; // Ethereum mainnet

    println!("📝 Fetching source code for: {}", test_address);
    println!("   Chain ID: {}", chain_id);
    println!();

    match fetch_source(chain_id, test_address).await {
        Ok(bundle) => {
            println!("✅ Source code fetched successfully!\n");
            println!("📊 Contract Information:");
            println!("   Compiler Version: {}", bundle.compiler_version);
            println!("   Number of Files: {}", bundle.files.len());
            println!("   ABI Length: {} bytes", bundle.abi.len());
            println!();

            println!("📁 Files:");
            for (filename, source) in bundle.files.iter() {
                let lines = source.lines().count();
                let chars = source.len();
                println!("   - {} ({} lines, {} chars)", filename, lines, chars);
            }
            println!();

            println!("🎯 Test Complete!");
        }
        Err(e) => {
            eprintln!("❌ Failed to fetch source code: {}", e);
            eprintln!();
            eprintln!("💡 Troubleshooting:");
            eprintln!("   1. Check ETHERSCAN_API_KEY is set in .env");
            eprintln!("   2. Ensure the contract address is verified on Etherscan");
            eprintln!("   3. Verify internet connection");
            eprintln!("   4. Check Etherscan API rate limits");
            return Err(e);
        }
    }

    Ok(())
}
