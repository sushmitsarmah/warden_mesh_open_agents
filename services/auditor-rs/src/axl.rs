use auditor::messages::Target;

pub struct TargetStream;

impl TargetStream {
    pub async fn next(&mut self) -> Option<Target> {
        // Placeholder: in production this connects to AXL mesh.
        None
    }
}

pub async fn subscribe_targets() -> TargetStream {
    TargetStream
}
