# MixOS Recovery Model (MRM)

## Overview

MixOS Recovery Model (MRM) adalah custom small language model yang di-embed langsung ke dalam MixOS untuk menangani boot problems, system recovery, dan self-healing. Model ini didesain khusus untuk environment OS, bukan general-purpose LLM.

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│   "A tiny brain embedded in the OS that understands only       │
│    one thing deeply: how to fix itself"                        │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 🎯 Design Goals

| Goal | Target | Rationale |
|------|--------|-----------|
| **Size** | < 100MB | Fit in initramfs |
| **RAM** | < 256MB | Work on minimal systems |
| **Inference** | < 100ms | Real-time response |
| **Accuracy** | > 95% | Reliable recovery |
| **Scope** | OS-only | No bloat, no hallucination |

---

## 🏗️ Model Architecture

### High-Level Design

```
┌─────────────────────────────────────────────────────────────────┐
│                    MRM Architecture                             │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Input                    Model                    Output       │
│  ─────                    ─────                    ──────       │
│                                                                 │
│  ┌──────────┐      ┌─────────────────┐      ┌──────────────┐   │
│  │  Error   │      │                 │      │   Action     │   │
│  │  Context │─────▶│   Transformer   │─────▶│   Sequence   │   │
│  │  State   │      │   (50M params)  │      │   + Params   │   │
│  └──────────┘      └─────────────────┘      └──────────────┘   │
│                                                                 │
│  Format:                                                        │
│  <error>...</error>        Encoder-only         <action>...</>  │
│  <context>...</context>    + Classification     <params>...</>  │
│  <state>...</state>        Head                 <confidence>N</>│
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Detailed Architecture

```
MRM-50M Architecture
====================

┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  Embedding Layer                                                │
│  ├── Vocabulary: 8,192 tokens (OS-specific)                    │
│  ├── Embedding dim: 512                                        │
│  └── Max sequence: 512 tokens                                  │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Transformer Blocks (x8)                                        │
│  ├── Multi-head attention (8 heads)                            │
│  ├── Feed-forward dim: 2048                                    │
│  ├── Layer norm (pre-norm)                                     │
│  ├── Dropout: 0.1                                              │
│  └── Activation: SwiGLU                                        │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  Output Heads                                                   │
│  ├── Action classifier (32 action types)                       │
│  ├── Parameter generator (sequence)                            │
│  └── Confidence estimator (0-1)                                │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘

Parameter Count:
- Embeddings: 8192 × 512 = 4.2M
- Transformer: 8 × (512² × 4 + 512 × 2048 × 2) = ~42M
- Output heads: ~4M
- Total: ~50M parameters

Quantized Size:
- FP32: 200MB
- FP16: 100MB
- INT8: 50MB
- INT4: 25MB (target)
```

### Alternative: Smaller MRM-20M

```
MRM-20M Architecture (Ultra-minimal)
====================================

- Vocabulary: 4,096 tokens
- Embedding dim: 256
- Transformer blocks: 6
- Attention heads: 4
- Feed-forward dim: 1024
- Max sequence: 256 tokens

Parameter Count: ~20M
INT4 Size: ~10MB
```

---

## 📚 Dataset Specification

### Dataset Structure

```
mixos-recovery-dataset/
├── raw/                          # Raw collected data
│   ├── kernel_panics/
│   ├── mount_failures/
│   ├── service_crashes/
│   ├── hardware_issues/
│   └── config_errors/
│
├── processed/                    # Cleaned & formatted
│   ├── train.jsonl              # 80% - Training
│   ├── valid.jsonl              # 10% - Validation
│   └── test.jsonl               # 10% - Testing
│
├── synthetic/                    # Generated examples
│   ├── augmented.jsonl
│   └── edge_cases.jsonl
│
└── metadata/
    ├── taxonomy.json            # Error taxonomy
    ├── actions.json             # Action definitions
    └── stats.json               # Dataset statistics
```

### Data Format (JSONL)

```jsonl
{
  "id": "kp_001",
  "category": "kernel_panic",
  "subcategory": "null_pointer",
  "input": {
    "error": "Kernel panic - not syncing: Attempted to kill init! exitcode=0x00000009",
    "context": {
      "kernel_version": "6.1.0-mixos",
      "last_service": "mix-agent",
      "boot_stage": "init",
      "uptime_seconds": 12
    },
    "state": {
      "memory_available": true,
      "root_mounted": true,
      "network_up": false,
      "services_started": ["broker", "pkgmgr"]
    }
  },
  "output": {
    "diagnosis": "Init process (PID 1) was killed, likely due to service dependency failure",
    "root_cause": "mix-agent failed to start due to missing python interpreter",
    "actions": [
      {
        "action": "disable_service",
        "params": {"service": "mix-agent"},
        "order": 1
      },
      {
        "action": "reboot",
        "params": {"mode": "normal"},
        "order": 2
      }
    ],
    "confidence": 0.92,
    "severity": "critical"
  },
  "metadata": {
    "source": "real_incident",
    "verified": true,
    "added_date": "2026-01-15"
  }
}
```

### Dataset Categories & Targets

```yaml
Categories:
  kernel_panic:
    description: "Kernel crashes and panics"
    target_count: 2000
    subcategories:
      - null_pointer_dereference
      - stack_overflow
      - out_of_memory
      - init_killed
      - driver_fault
      - filesystem_corruption
    
  mount_failure:
    description: "Filesystem mount issues"
    target_count: 1500
    subcategories:
      - device_not_found
      - filesystem_corrupted
      - wrong_fstype
      - permission_denied
      - busy_device
      - missing_module
    
  service_crash:
    description: "Service startup/runtime failures"
    target_count: 2000
    subcategories:
      - dependency_missing
      - config_invalid
      - port_in_use
      - permission_denied
      - resource_exhausted
      - timeout
    
  hardware_issue:
    description: "Hardware detection/driver problems"
    target_count: 1000
    subcategories:
      - device_not_detected
      - driver_not_loaded
      - firmware_missing
      - incompatible_hardware
      - resource_conflict
    
  config_error:
    description: "Configuration file problems"
    target_count: 1500
    subcategories:
      - syntax_error
      - invalid_value
      - missing_required
      - type_mismatch
      - circular_dependency

Total Target: 8000 examples
```

### Dataset Template

```json
{
  "$schema": "https://mixos.dev/schemas/recovery-dataset-v1.json",
  "version": "1.0",
  "example": {
    "id": "string (unique identifier)",
    "category": "enum (kernel_panic|mount_failure|service_crash|hardware_issue|config_error)",
    "subcategory": "string",
    "input": {
      "error": "string (raw error message/log)",
      "context": {
        "kernel_version": "string",
        "boot_stage": "enum (bootloader|kernel|init|services|ready)",
        "last_action": "string (optional)",
        "uptime_seconds": "number",
        "custom_fields": "object (optional)"
      },
      "state": {
        "memory_available": "boolean",
        "root_mounted": "boolean",
        "network_up": "boolean",
        "services_started": "array of strings",
        "custom_state": "object (optional)"
      }
    },
    "output": {
      "diagnosis": "string (human-readable explanation)",
      "root_cause": "string (technical root cause)",
      "actions": [
        {
          "action": "string (action_id from actions.json)",
          "params": "object",
          "order": "number",
          "fallback": "string (optional, alternative action)"
        }
      ],
      "confidence": "number (0.0-1.0)",
      "severity": "enum (info|warning|error|critical)"
    },
    "metadata": {
      "source": "enum (real_incident|synthetic|documentation|expert)",
      "verified": "boolean",
      "added_date": "string (ISO date)",
      "tags": "array of strings (optional)"
    }
  }
}
```

---

## ⚙️ Training Configuration

### Base Config

```yaml
# config/training/base.yaml

model:
  architecture: "mrm-50m"
  vocab_size: 8192
  hidden_size: 512
  num_layers: 8
  num_heads: 8
  intermediate_size: 2048
  max_position_embeddings: 512
  dropout: 0.1
  activation: "swiglu"
  
tokenizer:
  type: "bpe"
  vocab_size: 8192
  special_tokens:
    - "<pad>"
    - "<unk>"
    - "<bos>"
    - "<eos>"
    - "<error>"
    - "</error>"
    - "<context>"
    - "</context>"
    - "<state>"
    - "</state>"
    - "<action>"
    - "</action>"
    - "<params>"
    - "</params>"
    - "<confidence>"
    - "</confidence>"

training:
  batch_size: 32
  learning_rate: 3e-4
  weight_decay: 0.01
  warmup_steps: 1000
  max_steps: 50000
  gradient_accumulation: 4
  fp16: true
  
  optimizer:
    type: "adamw"
    betas: [0.9, 0.95]
    eps: 1e-8
    
  scheduler:
    type: "cosine"
    min_lr: 1e-5
    
  checkpointing:
    save_steps: 1000
    keep_last: 5

evaluation:
  eval_steps: 500
  metrics:
    - accuracy
    - action_f1
    - confidence_calibration
    
data:
  train_file: "data/processed/train.jsonl"
  valid_file: "data/processed/valid.jsonl"
  test_file: "data/processed/test.jsonl"
  max_length: 512
  
output:
  dir: "outputs/mrm-50m"
  logging_steps: 100
```

### Quantization Config

```yaml
# config/quantization/int4.yaml

quantization:
  method: "gptq"  # or "awq", "ggml"
  bits: 4
  group_size: 128
  damp_percent: 0.1
  
  calibration:
    dataset: "data/processed/valid.jsonl"
    num_samples: 256
    seq_length: 512
    
  output:
    format: "gguf"  # for llama.cpp compatibility
    path: "outputs/mrm-50m-int4.gguf"
```

### Inference Config

```yaml
# config/inference/production.yaml

inference:
  model_path: "/opt/mixos/models/mrm-50m-int4.gguf"
  
  runtime:
    backend: "ggml"  # lightweight C++ inference
    threads: 2
    batch_size: 1
    
  generation:
    max_tokens: 128
    temperature: 0.1  # low for deterministic output
    top_p: 0.9
    repetition_penalty: 1.1
    
  memory:
    context_size: 512
    kv_cache_type: "f16"
    
  timeout:
    inference_ms: 100
    total_ms: 500
```

---

## 🎬 Action Definitions

```json
{
  "version": "1.0",
  "actions": {
    "reboot": {
      "id": "reboot",
      "description": "Reboot the system",
      "params": {
        "mode": {
          "type": "enum",
          "values": ["normal", "recovery", "safe"],
          "required": true
        },
        "delay_seconds": {
          "type": "number",
          "default": 0
        }
      },
      "risk_level": "medium",
      "requires_confirmation": true
    },
    
    "remount_filesystem": {
      "id": "remount_filesystem",
      "description": "Remount a filesystem with different options",
      "params": {
        "path": {"type": "string", "required": true},
        "options": {"type": "string", "default": "rw"},
        "fstype": {"type": "string", "required": false}
      },
      "risk_level": "low",
      "requires_confirmation": false
    },
    
    "fsck": {
      "id": "fsck",
      "description": "Run filesystem check",
      "params": {
        "device": {"type": "string", "required": true},
        "auto_fix": {"type": "boolean", "default": false}
      },
      "risk_level": "medium",
      "requires_confirmation": true
    },
    
    "load_module": {
      "id": "load_module",
      "description": "Load a kernel module",
      "params": {
        "module": {"type": "string", "required": true},
        "params": {"type": "string", "required": false}
      },
      "risk_level": "low",
      "requires_confirmation": false
    },
    
    "unload_module": {
      "id": "unload_module",
      "description": "Unload a kernel module",
      "params": {
        "module": {"type": "string", "required": true},
        "force": {"type": "boolean", "default": false}
      },
      "risk_level": "medium",
      "requires_confirmation": true
    },
    
    "restart_service": {
      "id": "restart_service",
      "description": "Restart a system service",
      "params": {
        "service": {"type": "string", "required": true},
        "clean_state": {"type": "boolean", "default": false}
      },
      "risk_level": "low",
      "requires_confirmation": false
    },
    
    "disable_service": {
      "id": "disable_service",
      "description": "Disable a service from starting",
      "params": {
        "service": {"type": "string", "required": true},
        "temporary": {"type": "boolean", "default": true}
      },
      "risk_level": "low",
      "requires_confirmation": false
    },
    
    "rollback_package": {
      "id": "rollback_package",
      "description": "Rollback a package to previous version",
      "params": {
        "package": {"type": "string", "required": true},
        "version": {"type": "string", "required": false}
      },
      "risk_level": "medium",
      "requires_confirmation": true
    },
    
    "restore_config": {
      "id": "restore_config",
      "description": "Restore configuration from backup",
      "params": {
        "config_path": {"type": "string", "required": true},
        "backup_id": {"type": "string", "required": false}
      },
      "risk_level": "low",
      "requires_confirmation": false
    },
    
    "emergency_shell": {
      "id": "emergency_shell",
      "description": "Drop to emergency shell for manual intervention",
      "params": {
        "message": {"type": "string", "required": false}
      },
      "risk_level": "high",
      "requires_confirmation": true
    },
    
    "network_reset": {
      "id": "network_reset",
      "description": "Reset network configuration",
      "params": {
        "interface": {"type": "string", "default": "all"}
      },
      "risk_level": "medium",
      "requires_confirmation": true
    },
    
    "clear_cache": {
      "id": "clear_cache",
      "description": "Clear system caches",
      "params": {
        "cache_type": {
          "type": "enum",
          "values": ["all", "package", "build", "dns"],
          "default": "all"
        }
      },
      "risk_level": "low",
      "requires_confirmation": false
    },
    
    "repair_store": {
      "id": "repair_store",
      "description": "Repair the content-addressable store",
      "params": {
        "verify_only": {"type": "boolean", "default": true}
      },
      "risk_level": "medium",
      "requires_confirmation": true
    },
    
    "wait_and_retry": {
      "id": "wait_and_retry",
      "description": "Wait for a condition and retry",
      "params": {
        "condition": {"type": "string", "required": true},
        "timeout_seconds": {"type": "number", "default": 30},
        "retry_action": {"type": "string", "required": true}
      },
      "risk_level": "low",
      "requires_confirmation": false
    },
    
    "log_and_continue": {
      "id": "log_and_continue",
      "description": "Log the issue and continue boot",
      "params": {
        "severity": {
          "type": "enum",
          "values": ["info", "warning", "error"],
          "default": "warning"
        },
        "message": {"type": "string", "required": true}
      },
      "risk_level": "low",
      "requires_confirmation": false
    },
    
    "notify_user": {
      "id": "notify_user",
      "description": "Display notification to user",
      "params": {
        "title": {"type": "string", "required": true},
        "message": {"type": "string", "required": true},
        "severity": {
          "type": "enum",
          "values": ["info", "warning", "error"],
          "default": "info"
        }
      },
      "risk_level": "low",
      "requires_confirmation": false
    }
  }
}
```

---

## 🔄 Knowledge Update Cycle

### Update Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Knowledge Update Pipeline                    │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │   Collect   │───▶│   Process   │───▶│   Train     │         │
│  │   (Daily)   │    │   (Weekly)  │    │  (Monthly)  │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│        │                  │                  │                  │
│        ▼                  ▼                  ▼                  │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐         │
│  │  Telemetry  │    │  Validate   │    │  Quantize   │         │
│  │  Incidents  │    │  & Clean    │    │  & Package  │         │
│  │  Feedback   │    │             │    │             │         │
│  └─────────────┘    └─────────────┘    └─────────────┘         │
│                                              │                  │
│                                              ▼                  │
│                                        ┌─────────────┐         │
│                                        │   Deploy    │         │
│                                        │  (Release)  │         │
│                                        └─────────────┘         │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Update Cycle Phases

```yaml
# Knowledge Update Cycle Configuration

collection:
  frequency: "daily"
  sources:
    - type: "telemetry"
      description: "Anonymous error reports from MixOS installations"
      consent_required: true
      data_collected:
        - error_messages
        - system_state
        - recovery_actions_taken
        - success_rate
        
    - type: "incidents"
      description: "Manually reported issues from GitHub/forums"
      review_required: true
      
    - type: "feedback"
      description: "User feedback on recovery suggestions"
      fields:
        - was_helpful: boolean
        - correct_action: boolean
        - alternative_action: string (optional)
        
    - type: "synthetic"
      description: "Generated edge cases from documentation"
      generation_method: "template_based"
      
  storage:
    location: "s3://mixos-ml/raw-data/"
    retention_days: 365
    encryption: true

processing:
  frequency: "weekly"
  steps:
    - name: "deduplication"
      method: "semantic_similarity"
      threshold: 0.95
      
    - name: "validation"
      checks:
        - schema_valid
        - action_exists
        - params_valid
        
    - name: "labeling"
      method: "semi_automatic"
      confidence_threshold: 0.8
      manual_review_below: 0.8
      
    - name: "augmentation"
      techniques:
        - paraphrase_error_messages
        - vary_context_values
        - combine_related_errors
        
  output:
    format: "jsonl"
    location: "s3://mixos-ml/processed/"

training:
  frequency: "monthly"
  trigger:
    min_new_examples: 500
    or_max_days: 30
    
  process:
    - name: "merge_datasets"
      keep_ratio:
        existing: 0.7
        new: 0.3
        
    - name: "train"
      config: "config/training/base.yaml"
      epochs: 3
      early_stopping:
        patience: 5
        metric: "action_f1"
        
    - name: "evaluate"
      metrics:
        - accuracy
        - action_f1
        - confidence_calibration
        - regression_check
      min_thresholds:
        accuracy: 0.95
        action_f1: 0.90
        
    - name: "quantize"
      config: "config/quantization/int4.yaml"
      
    - name: "benchmark"
      tests:
        - inference_speed
        - memory_usage
        - accuracy_post_quantization

deployment:
  frequency: "per_release"
  process:
    - name: "staging"
      duration_days: 7
      rollout_percentage: 5
      
    - name: "canary"
      duration_days: 7
      rollout_percentage: 20
      
    - name: "production"
      rollout: "gradual"
      duration_days: 14
      
  rollback:
    trigger:
      error_rate_increase: 0.1
      user_complaints: 10
    automatic: true
    
  versioning:
    format: "mrm-{major}.{minor}.{patch}"
    changelog: true
    model_card: true
```

### Version Management

```
Model Versions:
├── mrm-1.0.0 (initial release)
│   ├── mrm-1.0.0-fp16.gguf (100MB)
│   ├── mrm-1.0.0-int8.gguf (50MB)
│   └── mrm-1.0.0-int4.gguf (25MB)
│
├── mrm-1.1.0 (monthly update)
│   ├── +500 new examples
│   ├── +3 new action types
│   └── improved mount_failure accuracy
│
├── mrm-1.2.0 (monthly update)
│   └── ...
│
└── mrm-2.0.0 (architecture change)
    ├── new tokenizer
    ├── expanded vocabulary
    └── breaking changes
```

### Telemetry Schema

```json
{
  "telemetry_event": {
    "event_id": "uuid",
    "timestamp": "ISO8601",
    "mixos_version": "string",
    "mrm_version": "string",
    
    "incident": {
      "category": "string",
      "error_hash": "string (anonymized)",
      "context_hash": "string (anonymized)"
    },
    
    "recovery": {
      "suggested_actions": ["action_id"],
      "executed_actions": ["action_id"],
      "success": "boolean",
      "time_to_recovery_ms": "number"
    },
    
    "feedback": {
      "user_rating": "number (1-5, optional)",
      "was_correct": "boolean (optional)",
      "comment_hash": "string (optional, anonymized)"
    },
    
    "system": {
      "arch": "string",
      "memory_mb": "number",
      "boot_count": "number"
    }
  }
}
```

---

## 📊 Target Metrics

### Model Performance

| Metric | Target | Minimum |
|--------|--------|---------|
| Overall Accuracy | 97% | 95% |
| Action F1 Score | 95% | 90% |
| Confidence Calibration | < 0.05 ECE | < 0.10 ECE |
| False Positive Rate | < 2% | < 5% |
| Inference Latency (INT4) | < 50ms | < 100ms |
| Memory Usage | < 200MB | < 256MB |
| Model Size (INT4) | < 30MB | < 50MB |

### Per-Category Targets

| Category | Accuracy | F1 |
|----------|----------|-----|
| kernel_panic | 95% | 92% |
| mount_failure | 98% | 96% |
| service_crash | 97% | 95% |
| hardware_issue | 93% | 90% |
| config_error | 98% | 96% |

---

## 🛠️ Implementation Roadmap

### Phase 1: Data Collection (Month 1-2)
- [ ] Define complete taxonomy
- [ ] Create data collection pipeline
- [ ] Gather initial 2000 real examples
- [ ] Generate 3000 synthetic examples
- [ ] Validate and clean dataset

### Phase 2: Model Development (Month 2-3)
- [ ] Implement custom tokenizer
- [ ] Build model architecture
- [ ] Training pipeline setup
- [ ] Initial training run
- [ ] Evaluation framework

### Phase 3: Optimization (Month 3-4)
- [ ] Hyperparameter tuning
- [ ] Quantization experiments
- [ ] Inference optimization
- [ ] Memory optimization
- [ ] Benchmark suite

### Phase 4: Integration (Month 4-5)
- [ ] Integrate with mix-agent-early
- [ ] Action executor implementation
- [ ] Safety guardrails
- [ ] Testing in QEMU
- [ ] Documentation

### Phase 5: Deployment (Month 5-6)
- [ ] Telemetry system
- [ ] Update pipeline
- [ ] Staging environment
- [ ] Beta release
- [ ] Production release

---

## 📁 File Structure

```
mixos-recovery-model/
├── config/
│   ├── training/
│   │   ├── base.yaml
│   │   ├── mrm-50m.yaml
│   │   └── mrm-20m.yaml
│   ├── quantization/
│   │   ├── int4.yaml
│   │   └── int8.yaml
│   └── inference/
│       └── production.yaml
│
├── data/
│   ├── raw/
│   ├── processed/
│   ├── synthetic/
│   └── metadata/
│       ├── taxonomy.json
│       ├── actions.json
│       └── schema.json
│
├── src/
│   ├── model/
│   │   ├── architecture.py
│   │   ├── tokenizer.py
│   │   └── heads.py
│   ├── training/
│   │   ├── trainer.py
│   │   ├── dataset.py
│   │   └── metrics.py
│   ├── inference/
│   │   ├── engine.py
│   │   └── quantize.py
│   └── data/
│       ├── collector.py
│       ├── processor.py
│       └── augmentor.py
│
├── scripts/
│   ├── train.py
│   ├── evaluate.py
│   ├── quantize.py
│   └── export.py
│
├── tests/
│   ├── test_model.py
│   ├── test_inference.py
│   └── test_actions.py
│
└── docs/
    ├── MODEL_CARD.md
    └── CHANGELOG.md
```

---

## 🔗 Integration with MixOS

```
Boot Sequence with MRM:
=======================

1. Kernel loads
2. Init starts
3. mix-agent-early loads MRM
4. For each boot step:
   │
   ├── Success → Continue
   │
   └── Failure → 
       ├── Collect error + context + state
       ├── Query MRM
       ├── Get suggested actions
       ├── Execute actions (with safety checks)
       ├── Verify result
       └── Continue or escalate

Integration Points:
- /opt/mixos/models/mrm.gguf (model file)
- /etc/mixos/mrm.yaml (configuration)
- /var/log/mixos/mrm.log (inference logs)
- /var/lib/mixos/mrm/telemetry/ (telemetry data)
```

---

*Document Version: 1.0*
*Last Updated: 2026-01-11*
