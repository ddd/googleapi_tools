use std::fs::File;
use std::io::{BufRead, BufReader};
use std::collections::HashMap;
use std::env;
use tokio;
use reqwest;
use serde_json::{Value, json};
use async_channel::{bounded, Receiver};
use futures::future::join_all;

#[derive(Debug)]
struct ScopeCheck {
    app: String,
    sig: String,
    spatula: String,
    scope: String,
}

#[derive(Debug)]
struct ScopeResult {
    app: String,
    sig: String,
    spatula: String,
    approved_scopes: Vec<String>,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Read the Android clients JSON file
    let clients_file = File::open("../data/android_clients.json")?;
    let clients: Value = serde_json::from_reader(clients_file)?;

    // Read scopes from input file
    let args: Vec<String> = env::args().collect();
    if args.len() < 2 {
        eprintln!("Usage: {} <scope1> [scope2] [scope3] ...", args[0]);
        eprintln!("Example: {} https://www.googleapis.com/auth/calendar", args[0]);
        std::process::exit(1);
    }

    // Skip the first argument (program name) and collect the rest as scopes
    let scopes: Vec<String> = args[1..].to_vec();
    println!("Checking scopes: {:?}", scopes);
    
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
    let clients = clients.as_object().unwrap();
    for (app, data) in clients {
        for client in data.as_array().unwrap() {
            let client_obj = client.as_object().unwrap();
            for scope in &scopes {
                sender.send(ScopeCheck {
                    app: app.clone(),
                    sig: client_obj["sig"].as_str().unwrap().to_string(),
                    spatula: client_obj["spatula"].as_str().unwrap().to_string(),
                    scope: scope.clone(),
                }).await?;
            }
        }
    }

    // Close the sender when all tasks have been sent
    drop(sender);

    // Wait for all workers to complete and collect results
    let results: Vec<ScopeResult> = join_all(handles)
        .await
        .into_iter()
        .filter_map(|r| r.ok())
        .flatten()
        .collect();

    // Process results and build output JSON
    let mut output: HashMap<String, Vec<String>> = HashMap::new();
    for result in results {
        if !result.approved_scopes.is_empty() {
            let entry = output.entry(result.app).or_insert_with(Vec::new);
            for scope in result.approved_scopes {
                if !entry.contains(&scope) {
                    entry.push(scope);
                }
            }
        }
    }

    // Write the results to output/clients.json
    let output_file = File::create("output/clients.json")?;
    serde_json::to_writer_pretty(output_file, &output)?;

    println!("Results written to output/clients.json");

    Ok(())
}

async fn worker(receiver: Receiver<ScopeCheck>, token: String) -> Vec<ScopeResult> {
    let mut results = Vec::new();
    let mut current_app = String::new();
    let mut current_sig = String::new();
    let mut current_spatula = String::new();
    let mut approved_scopes = Vec::new();

    let client = reqwest::Client::new();

    while let Ok(task) = receiver.recv().await {
        if task.app != current_app || task.sig != current_sig {
            if !current_app.is_empty() && !approved_scopes.is_empty() {
                results.push(ScopeResult {
                    app: current_app,
                    sig: current_sig,
                    spatula: current_spatula,
                    approved_scopes: approved_scopes,
                });
            }
            current_app = task.app.clone();
            current_sig = task.sig.clone();
            current_spatula = task.spatula;
            approved_scopes = Vec::new();
        }

        let response = client
            .post("https://android.googleapis.com/auth")
            .header("User-Agent", "GoogleAuth/1.4")
            .header("Content-Type", "application/x-www-form-urlencoded")
            .body(format!(
                "app={}&service=oauth2:{}&client_sig={}&Token={}",
                task.app, task.scope, task.sig, token
            ))
            .send()
            .await;

        match response {
            Ok(resp) => {
                let text = resp.text().await.unwrap_or_default();
                if !text.contains("Error=RESTRICTED_CLIENT") && !text.contains("Error=UNREGISTERED_ON_API_CONSOLE") {
                    println!("Found approved scope: {} for app={}, sig={}", task.scope, task.app, task.sig);
                    approved_scopes.push(task.scope);
                }
            }
            Err(e) => {
                eprintln!("Request failed: {}. Skipping...", e);
            }
        }
    }

    // Add the last result if it has approved scopes
    if !current_app.is_empty() && !approved_scopes.is_empty() {
        results.push(ScopeResult {
            app: current_app,
            sig: current_sig,
            spatula: current_spatula,
            approved_scopes,
        });
    }

    results
}

fn read_lines(filename: &str) -> std::io::Result<Vec<String>> {
    let file = File::open(filename)?;
    let reader = BufReader::new(file);
    reader.lines().collect()
}