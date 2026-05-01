use std::io::{self, Read};
use auditor::messages::Target;

fn main() {
    let mut buf = String::new();
    io::stdin().read_to_string(&mut buf).expect("read stdin");
    let target: Target = serde_json::from_str(&buf).expect("parse JSON");
    assert_eq!(target.id, "test-target-001");
    println!("roundtrip OK: id={}", target.id);
}
