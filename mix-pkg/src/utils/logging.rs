// Logging utilities for mix-pkg
use std::fs::{self, File, OpenOptions};
use std::io::Write;
use std::path::{Path, PathBuf};
use std::sync::Mutex;
use colored::Colorize;

/// Log level
#[derive(Debug, Clone, Copy, PartialEq, Eq, PartialOrd, Ord)]
pub enum LogLevel {
    Debug,
    Info,
    Warning,
    Error,
}

impl LogLevel {
    pub fn from_str(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "debug" => Self::Debug,
            "info" => Self::Info,
            "warning" | "warn" => Self::Warning,
            "error" => Self::Error,
            _ => Self::Info,
        }
    }
    
    pub fn as_str(&self) -> &'static str {
        match self {
            Self::Debug => "DEBUG",
            Self::Info => "INFO",
            Self::Warning => "WARN",
            Self::Error => "ERROR",
        }
    }
    
    pub fn colored_prefix(&self) -> String {
        match self {
            Self::Debug => "DEBUG".dimmed().to_string(),
            Self::Info => "INFO".blue().to_string(),
            Self::Warning => "WARN".yellow().to_string(),
            Self::Error => "ERROR".red().to_string(),
        }
    }
}

/// Logger configuration
pub struct LoggerConfig {
    pub level: LogLevel,
    pub file_path: Option<PathBuf>,
    pub console_output: bool,
    pub timestamps: bool,
    pub colors: bool,
}

impl Default for LoggerConfig {
    fn default() -> Self {
        Self {
            level: LogLevel::Info,
            file_path: None,
            console_output: true,
            timestamps: true,
            colors: true,
        }
    }
}

/// Global logger
pub struct Logger {
    config: LoggerConfig,
    file: Option<Mutex<File>>,
}

impl Logger {
    pub fn new(config: LoggerConfig) -> Self {
        let file = config.file_path.as_ref().and_then(|path| {
            if let Some(parent) = path.parent() {
                let _ = fs::create_dir_all(parent);
            }
            
            OpenOptions::new()
                .create(true)
                .append(true)
                .open(path)
                .ok()
                .map(Mutex::new)
        });
        
        Self { config, file }
    }
    
    pub fn log(&self, level: LogLevel, message: &str) {
        if level < self.config.level {
            return;
        }
        
        let timestamp = if self.config.timestamps {
            format!("[{}] ", current_timestamp())
        } else {
            String::new()
        };
        
        // Console output
        if self.config.console_output {
            let prefix = if self.config.colors {
                level.colored_prefix()
            } else {
                level.as_str().to_string()
            };
            
            eprintln!("{}[{}] {}", timestamp, prefix, message);
        }
        
        // File output
        if let Some(ref file) = self.file {
            if let Ok(mut f) = file.lock() {
                let _ = writeln!(f, "{}[{}] {}", timestamp, level.as_str(), message);
            }
        }
    }
    
    pub fn debug(&self, message: &str) {
        self.log(LogLevel::Debug, message);
    }
    
    pub fn info(&self, message: &str) {
        self.log(LogLevel::Info, message);
    }
    
    pub fn warning(&self, message: &str) {
        self.log(LogLevel::Warning, message);
    }
    
    pub fn error(&self, message: &str) {
        self.log(LogLevel::Error, message);
    }
}

/// Progress indicator
pub struct Progress {
    total: u64,
    current: u64,
    message: String,
    bar_width: usize,
    show_percentage: bool,
    show_speed: bool,
    start_time: std::time::Instant,
}

impl Progress {
    pub fn new(total: u64, message: &str) -> Self {
        Self {
            total,
            current: 0,
            message: message.to_string(),
            bar_width: 40,
            show_percentage: true,
            show_speed: true,
            start_time: std::time::Instant::now(),
        }
    }
    
    pub fn update(&mut self, current: u64) {
        self.current = current;
        self.render();
    }
    
    pub fn increment(&mut self, amount: u64) {
        self.current += amount;
        self.render();
    }
    
    pub fn finish(&self) {
        eprintln!();
    }
    
    fn render(&self) {
        let percentage = if self.total > 0 {
            (self.current as f64 / self.total as f64 * 100.0) as u32
        } else {
            0
        };
        
        let filled = (self.bar_width as f64 * self.current as f64 / self.total.max(1) as f64) as usize;
        let empty = self.bar_width - filled;
        
        let bar = format!(
            "[{}{}]",
            "█".repeat(filled).cyan(),
            "░".repeat(empty)
        );
        
        let speed = if self.show_speed {
            let elapsed = self.start_time.elapsed().as_secs_f64();
            if elapsed > 0.0 {
                let speed = self.current as f64 / elapsed;
                format!(" {}/s", format_size(speed as u64))
            } else {
                String::new()
            }
        } else {
            String::new()
        };
        
        let percentage_str = if self.show_percentage {
            format!(" {:3}%", percentage)
        } else {
            String::new()
        };
        
        eprint!(
            "\r{} {} {}/{}{}{}",
            self.message,
            bar,
            format_size(self.current),
            format_size(self.total),
            percentage_str,
            speed
        );
    }
}

/// Spinner for indeterminate progress
pub struct Spinner {
    message: String,
    frames: Vec<&'static str>,
    current_frame: usize,
    active: bool,
}

impl Spinner {
    pub fn new(message: &str) -> Self {
        Self {
            message: message.to_string(),
            frames: vec!["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"],
            current_frame: 0,
            active: true,
        }
    }
    
    pub fn tick(&mut self) {
        if !self.active {
            return;
        }
        
        let frame = self.frames[self.current_frame];
        eprint!("\r{} {}", frame.cyan(), self.message);
        
        self.current_frame = (self.current_frame + 1) % self.frames.len();
    }
    
    pub fn finish(&mut self, success: bool) {
        self.active = false;
        let icon = if success {
            "✓".green()
        } else {
            "✗".red()
        };
        eprintln!("\r{} {}", icon, self.message);
    }
}

/// Format size in human-readable format
pub fn format_size(bytes: u64) -> String {
    const KB: u64 = 1024;
    const MB: u64 = KB * 1024;
    const GB: u64 = MB * 1024;
    
    if bytes >= GB {
        format!("{:.2} GB", bytes as f64 / GB as f64)
    } else if bytes >= MB {
        format!("{:.2} MB", bytes as f64 / MB as f64)
    } else if bytes >= KB {
        format!("{:.2} KB", bytes as f64 / KB as f64)
    } else {
        format!("{} B", bytes)
    }
}

/// Format duration in human-readable format
pub fn format_duration(secs: u64) -> String {
    if secs >= 3600 {
        format!("{}h {}m", secs / 3600, (secs % 3600) / 60)
    } else if secs >= 60 {
        format!("{}m {}s", secs / 60, secs % 60)
    } else {
        format!("{}s", secs)
    }
}

/// Get current timestamp string
fn current_timestamp() -> String {
    use std::time::{SystemTime, UNIX_EPOCH};
    
    let duration = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap_or_default();
    
    let secs = duration.as_secs();
    let hours = (secs % 86400) / 3600;
    let minutes = (secs % 3600) / 60;
    let seconds = secs % 60;
    
    format!("{:02}:{:02}:{:02}", hours, minutes, seconds)
}

/// Print success message
pub fn print_success(message: &str) {
    println!("{} {}", "✓".green(), message);
}

/// Print error message
pub fn print_error(message: &str) {
    eprintln!("{} {}", "✗".red(), message);
}

/// Print warning message
pub fn print_warning(message: &str) {
    eprintln!("{} {}", "⚠".yellow(), message);
}

/// Print info message
pub fn print_info(message: &str) {
    println!("{} {}", "→".cyan(), message);
}

#[cfg(test)]
mod tests {
    use super::*;
    
    #[test]
    fn test_format_size() {
        assert_eq!(format_size(500), "500 B");
        assert_eq!(format_size(1024), "1.00 KB");
        assert_eq!(format_size(1024 * 1024), "1.00 MB");
        assert_eq!(format_size(1024 * 1024 * 1024), "1.00 GB");
    }
    
    #[test]
    fn test_format_duration() {
        assert_eq!(format_duration(30), "30s");
        assert_eq!(format_duration(90), "1m 30s");
        assert_eq!(format_duration(3700), "1h 1m");
    }
    
    #[test]
    fn test_log_level_ordering() {
        assert!(LogLevel::Debug < LogLevel::Info);
        assert!(LogLevel::Info < LogLevel::Warning);
        assert!(LogLevel::Warning < LogLevel::Error);
    }
}
