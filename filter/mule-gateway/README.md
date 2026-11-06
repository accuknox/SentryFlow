# SentryFlow Mule 4 Policy — Build & Publish Guide

This directory contains the Mule 4 custom policy that forwards API telemetry
from Anypoint Mule Gateway to SentryFlow.

## Directory Structure

```
filter/mule-gateway/
├── policy-definition/
│   ├── sentryflow-policy.json   # JSON Schema — defines API Manager UI
│   └── sentryflow-policy.yaml   # Policy metadata (category, config fields)
└── policy-implementation/
    ├── pom.xml                  # Maven build
    ├── mule-artifact.json       # Mule artifact descriptor
    └── src/main/mule/
        └── sentryflow-mule-policy.xml  # Policy flow logic
```

## Quick Start

### 1. Build the JAR

```shell
cd policy-implementation
mvn clean package
# → target/sentryflow-mule-policy-1.0.0-mule-policy.jar
```

### 2. Publish to Anypoint Exchange

Follow the full guide in [`docs/mule-gateway-integration.md`](../../docs/mule-gateway-integration.md)
for step-by-step instructions on uploading to Exchange and applying via API Manager.

## Requirements

- Java 11+
- Maven 3.6+
- Anypoint Platform account
- Mule Runtime 4.3.0+
