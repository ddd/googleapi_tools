use std::fs::File;
use std::io::{BufRead, BufReader};
use std::collections::HashMap;
use std::env;
use tokio;
use reqwest;
use async_channel::{bounded, Sender, Receiver};
use futures::future::join_all;
use std::path::Path;
use std::io::Write;
use prost::Message;
use base64::{Engine as _, engine::general_purpose};

const MAX_RETRIES: u32 = 10;

include!(concat!(env!("OUT_DIR"), "/spatula.rs"));

#[derive(Debug)]
struct Task {
    app: String,
    sig: String,
}

#[derive(Debug)]
struct TaskResult {
    app: String,
    sig: String,
    spatula: String,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Read input files
    let apps = read_lines("input/packages.txt")?;
    let signatures = read_lines("input/sig.txt")?;

    let token = env::var("ANDROID_REFRESH_TOKEN").expect("ANDROID_REFRESH_TOKEN must be set");

    // Get worker count from environment variable or use a default
    let worker_count = env::var("WORKER_COUNT").unwrap_or_else(|_| "50".to_string()).parse::<usize>()?;

    // Create a channel for task distribution
    let (sender, receiver) = bounded(worker_count * 2);

    // Spawn worker tasks
    let mut handles = vec![];
    for _ in 0..worker_count {
        let receiver = receiver.clone();
        let worker_token = token.clone();
        let handle = tokio::spawn(async move {
            worker(receiver, worker_token).await
        });
        handles.push(handle);
    }

    // Distribute tasks to workers
    for app in &apps {
        for sig in &signatures {
            sender.send(Task {
                app: app.clone(),
                sig: sig.clone(),
            }).await?;
        }
    }

    // Close the sender when all tasks have been sent
    drop(sender);

    // Wait for all workers to complete and collect results
    let results: Vec<TaskResult> = join_all(handles)
        .await
        .into_iter()
        .filter_map(|r| r.ok())
        .flatten()
        .collect();

    // Process results and build output JSON
    let mut output: HashMap<String, Vec<HashMap<String, String>>> = HashMap::new();
    for result in results {
        output.entry(result.app)
            .or_insert_with(Vec::new)
            .push(HashMap::from([
                ("sig".to_string(), result.sig),
                ("spatula".to_string(), result.spatula),
            ]));
    }

    // Write the result as JSON to data/android_clients.json
    let output_path = Path::new("../data/android_clients.json");
    let mut file = File::create(output_path)?;
    let json = serde_json::to_string_pretty(&output)?;
    file.write_all(json.as_bytes())?;

    println!("Results written to data/android_clients.json");

    Ok(())
}

async fn worker(receiver: Receiver<Task>, token: String) -> Vec<TaskResult> {
    let mut results = Vec::new();
    while let Ok(task) = receiver.recv().await {
        let success = check_app_signature(&token, &task.app, &task.sig).await;
        if success {
            let spatula = generate_spatula(&task.app, &task.sig);
            results.push(TaskResult {
                app: task.app,
                sig: task.sig,
                spatula
            });
        }
    }
    results
}

fn generate_spatula(app: &str, sig: &str) -> String {
    // Convert hex signature to bytes
    let sig_bytes = hex::decode(sig).expect("Invalid hex signature");
    
    // Create AppInfo
    let app_info = AppInfo {
        package_name: app.to_string(),
        signature: general_purpose::STANDARD.encode(&sig_bytes),
    };

    // Create SpatulaInner
    let spatula_inner = SpatulaInner {
        app_info: Some(app_info),
        droidguard_response: 3959931537119515576,
    };

    // Encode SpatulaInner to protobuf
    let mut buf = Vec::new();
    spatula_inner.encode(&mut buf).expect("Failed to encode protobuf");

    // Base64 encode the protobuf
    general_purpose::STANDARD.encode(buf)
}

async fn check_app_signature(token: &str, app: &str, sig: &str) -> bool {
    let client = reqwest::Client::new();

    for _ in 0..MAX_RETRIES {
        let response = client
            .post("https://android.googleapis.com/auth")
            .header("User-Agent", "GoogleAuth/1.4")
            .header("Content-Type", "application/x-www-form-urlencoded")
            .body(format!(
                "app={}&service=oauth2:https://www.googleapis.com/auth/peopleapi.readwrite&client_sig={}&Token={}",
                app, sig, token
            ))
            .send()
            .await;

        match response {
            Ok(resp) => {
                let text = resp.text().await.unwrap_or_default();
                if !text.contains("Error=UNREGISTERED_ON_API_CONSOLE") {
                    println!("Found: app={}, sig={}", app, sig);
                    return true;
                }
                return false;
            }
            Err(e) => {
                eprintln!("Request failed: {}. Retrying...", e);
                tokio::time::sleep(tokio::time::Duration::from_secs(1)).await;
            }
        }
    }
    false
}

fn read_lines(filename: &str) -> std::io::Result<Vec<String>> {
    let file = File::open(filename)?;
    let reader = BufReader::new(file);
    reader.lines().collect()
}