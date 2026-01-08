use anyhow::{Result, Context};
use serde::{Deserialize, Serialize};
use std::path::PathBuf;

#[derive(Debug, Serialize, Deserialize)]
pub struct Config {
    pub repositories: Vec<Repository>,
    #[serde(default)]
    pub cache_dir: Option<PathBuf>,
    #[serde(default)]
    pub db_path: Option<PathBuf>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Repository {
    pub name: String,
    pub url: String,
    #[serde(default = "default_enabled")]
    pub enabled: bool,
}

fn default_enabled() -> bool {
    true
}

impl Default for Config {
    fn default() -> Self {
        Config {
            repositories: vec![
                Repository {
                    name: "core".to_string(),
                    url: "https://repo.mixos.dev/core".to_string(),
                    enabled: true,
                },
                Repository {
                    name: "dev".to_string(),
                    url: "https://repo.mixos.dev/dev".to_string(),
                    enabled: true,
                },
                Repository {
                    name: "ai".to_string(),
                    url: "https://repo.mixos.dev/ai".to_string(),
                    enabled: true,
                },
            ],
            cache_dir: None,
            db_path: None,
        }
    }
}

impl Config {
    pub fn load() -> Result<Self> {
        let config_paths = [
            PathBuf::from("/etc/mixos/repositories.toml"),
            dirs::config_dir()
                .unwrap_or_else(|| PathBuf::from("~/.config"))
                .join("mixos/repositories.toml"),
        ];
        
        for path in &config_paths {
            if path.exists() {
                let content = std::fs::read_to_string(path)
                    .context(format!("Failed to read config from {:?}", path))?;
                let config: Config = toml::from_str(&content)
                    .context("Failed to parse config")?;
                return Ok(config);
            }
        }
        
        // Return default config if no file found
        Ok(Config::default())
    }
    
    pub fn save(&self) -> Result<()> {
        let config_dir = PathBuf::from("/etc/mixos");
        std::fs::create_dir_all(&config_dir)?;
        
        let config_path = config_dir.join("repositories.toml");
        let content = toml::to_string_pretty(self)?;
        std::fs::write(&config_path, content)?;
        
        Ok(())
    }
    
    pub fn cache_dir(&self) -> PathBuf {
        self.cache_dir.clone().unwrap_or_else(|| {
            dirs::cache_dir()
                .unwrap_or_else(|| PathBuf::from("/var/cache"))
                .join("mix-pkg")
        })
    }
    
    pub fn db_path(&self) -> PathBuf {
        self.db_path.clone().unwrap_or_else(|| {
            PathBuf::from("/var/lib/mix-pkg/db")
        })
    }
}
