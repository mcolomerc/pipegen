
# `clean`

Clean up Docker resources and free up system resources used by PipeGen.

## Usage

```bash
pipegen clean
```

## Description

The `clean` command removes Docker containers, networks, and volumes created by PipeGen during pipeline runs and development. This helps reclaim disk space and ensures a fresh environment for future executions.

## Options

This command does not take any options.

## Example

```bash
pipegen clean
```

This will remove all PipeGen-related Docker resources.

## When to use
- After running pipelines or development stacks to free up resources
- Before starting a new pipeline to ensure a clean environment

## Related Commands
- [`run`](./run)
- [`deploy`](./deploy)
- [`dashboard`](./dashboard)
