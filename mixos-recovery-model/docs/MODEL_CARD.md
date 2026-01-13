# MRM Model Card

## Model Details

- **Name**: MixOS Recovery Model (MRM)
- **Version**: 2.0
- **Architecture**: Field Resonance (Being + Variants)
- **Size**: < 5MB (GGUF format)
- **Parameters**: ~200K effective

## Intended Use

- OS boot recovery
- System self-healing
- Error diagnosis and action recommendation

## Specifications

| Metric | Value |
|--------|-------|
| Variants | 64 |
| Domains | 8 |
| Field Resolution | 32³ |
| Inference Time | < 50ms |
| Memory Usage | < 200KB |

## Limitations

- OS-specific only (not general purpose)
- Requires structured input format
- Limited to predefined action set

## Training Data

- Synthetic examples from dataset.rs
- Real-world OS recovery scenarios (planned)

## Ethical Considerations

- Safety guardrails prevent destructive actions
- Requires confirmation for high-risk operations
- Fallback to emergency shell when uncertain
