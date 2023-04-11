use serde::{Deserialize, Serialize};

#[derive(Deserialize)]
#[serde()]
pub struct AdminLogin<'r> {
    pub password: &'r str,
}

#[derive(Serialize, Deserialize)]
#[serde()]
pub struct Stats {
    pub projects: u64,
    pub seen: u64,
    pub votes: u64,
    pub time: u64,
    pub avg_mu: f64,
    pub avg_sigma: f64,
    pub judges: u64,
}

impl Default for Stats {
    fn default() -> Self {
        Self {
            projects: 0,
            seen: 0,
            votes: 0,
            time: 0,
            avg_mu: 0.0,
            avg_sigma: 0.0,
            judges: 0,
        }
    }
}
