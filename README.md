# VictoriaLogs MCP Server

[![Latest Release](https://img.shields.io/github/v/release/VictoriaMetrics/mcp-victorialogs?sort=semver&label=&filter=!*-victorialogs&logo=github&labelColor=gray&color=gray&link=https%3A%2F%2Fgithub.com%2FVictoriaMetrics%2Fmcp-victorialogs%2Freleases%2Flatest)](https://github.com/VictoriaMetrics/mcp-victorialogs/releases)
![License](https://img.shields.io/github/license/VictoriaMetrics/mcp-victorialogs?labelColor=green&label=&link=https%3A%2F%2Fgithub.com%2FVictoriaMetrics%2Fmcp-victorialogs%2Fblob%2Fmain%2FLICENSE)
![Slack](https://img.shields.io/badge/Join-4A154B?logo=slack&link=https%3A%2F%2Fslack.victoriametrics.com)
![X](https://img.shields.io/twitter/follow/VictoriaMetrics?style=flat&label=Follow&color=black&logo=x&labelColor=black&link=https%3A%2F%2Fx.com%2FVictoriaMetrics)
![Reddit](https://img.shields.io/reddit/subreddit-subscribers/VictoriaMetrics?style=flat&label=Join&labelColor=red&logoColor=white&logo=reddit&link=https%3A%2F%2Fwww.reddit.com%2Fr%2FVictoriaMetrics)

The implementation of [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for [VictoriaLogs](https://docs.victoriametrics.com/victorialogs/).

This provides access to your VictoriaLogs instance and seamless integration with [VictoriaLogs APIs](https://docs.victoriametrics.com/victorialogs/querying/#http-api) and [documentation](https://docs.victoriametrics.com/victorialogs/).
It can give you a comprehensive interface for logs, observability, and debugging tasks related to your VictoriaLogs instances, enable advanced automation and interaction capabilities for engineers and tools.

## Features

This MCP server allows you to use almost all read-only APIs of VictoriaLogs, i.e. all functions available in [Web UI](https://docs.victoriametrics.com/victorialogs/querying/#web-ui):

- Querying logs and exploring logs data
- Showing parameters of your VictoriaLogs instance
- Listing available streams, fields, field values
- Query statistics for the logs as metrics
- UI with setup instructions and tools inspection on the root endpoint (only in Streamable HTTP mode)
 
In addition, the MCP server contains embedded up-to-date documentation and is able to search it without online access.

![image](./ui.png)

More details about the exact available tools and prompts can be found in the [Usage](#usage) section.

You can combine functionality of tools, docs search in your prompts and invent great usage scenarios for your VictoriaLogs instance.
And please note the fact that the quality of the MCP Server and its responses depends very much on the capabilities of your client and the quality of the model you are using.

You can also combine the MCP server with other observability or doc search related MCP Servers and get even more powerful results.

## Try without installation

There is a publicly available instance of the VictoriaMetrics MCP Server that you can use to test the features without installing it: 

```
https://play-vmlogs-mcp.victoriametrics.com/mcp
```

**Attention!** This URL is not supposed to be opened in a browser, it is intended to be used in MCP clients.

It's available in [Streamable HTTP Mode](#modes) mode and configured to work with [Public VictoriaLogs Playground](https://play-vmlogs.victoriametrics.com).

Here is example of configuration for [Claude Desktop](https://claude.ai/download):

![image](https://github.com/user-attachments/assets/938d9eb9-f188-42f1-8377-a283be454ac7)

## Requirements

- [VictoriaLogs](https://docs.victoriametrics.com/victorialogs/) instance: ([single-node](https://docs.victoriametrics.com/victorialogs/) or [cluster](https://docs.victoriametrics.com/victorialogs/cluster/))
- Go 1.26 or higher (if you want to build from source)

## Installation

### Go

```bash
go install github.com/VictoriaMetrics/mcp-victorialogs/cmd/mcp-victorialogs@latest
```

### Binaries

Just download the latest release from [Releases](https://github.com/VictoriaMetrics/mcp-victorialogs/releases) page and put it to your PATH.

Example for Linux x86_64 (note that other architectures and platforms are also available):

```bash
latest=$(curl -s https://api.github.com/repos/VictoriaMetrics/mcp-victorialogs/releases/latest | grep 'tag_name' | cut -d\" -f4)
wget https://github.com/VictoriaMetrics/mcp-victorialogs/releases/download/$latest/mcp-victorialogs_Linux_x86_64.tar.gz
tar axvf mcp-victorialogs_Linux_x86_64.tar.gz
```

### Docker

You can run VictoriaLogs MCP Server using Docker.

This is the easiest way to get started without needing to install Go or build from source.

```bash
docker run -d --name mcp-victorialogs \
  -e VL_INSTANCE_ENTRYPOINT=https://play-vmlogs.victoriametrics.com \
  -e MCP_SERVER_MODE=http \
  -e MCP_LISTEN_ADDR=:8081 \
  -p 8081:8081 \
  ghcr.io/victoriametrics/mcp-victorialogs
```

You should replace environment variables with your own parameters.

Note that the `MCP_SERVER_MODE=http` flag is used to enable Streamable HTTP mode. 
More details about server modes can be found in the [Configuration](#configuration) section.

See available docker images in [github registry](https://github.com/orgs/VictoriaMetrics/packages/container/package/mcp-victorialogs).

Also see [Using Docker instead of binary](#using-docker-instead-of-binary) section for more details about using Docker with MCP server with clients in stdio mode.


### Source Code

For building binary from source code you can use the following approach:

- Clone repo:

  ```bash
  git clone https://github.com/VictoriaMetrics/mcp-victorialogs.git
  cd mcp-victorialogs
  ```
- Build binary from cloned source code:

  ```bash
  make build
  # after that you can find binary mcp-victorialogs and copy this file to your PATH or run inplace
  ```
- Build image from cloned source code:

  ```bash
  docker build -t mcp-victorialogs .
  # after that you can use docker image mcp-victorialogs for running or pushing
  ```

### Helm

Check out [VictoriaLogs MCP Server Helm chart](https://docs.victoriametrics.com/helm/victoria-logs-mcp/) documentation for more details about installation using Helm.

## Configuration

MCP Server for VictoriaLogs is configured via environment variables:

| Variable                   | Description                                                                                                                                                                                                                                                         | Required | Default          | Allowed values                   |
|----------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|------------------|----------------------------------|
| `VL_INSTANCE_ENTRYPOINT`   | URL to VictoriaLogs instance                                                                                                                                                                                                                                        | Yes      | -                | -                                |
| `VL_INSTANCE_BEARER_TOKEN` | Authentication token for VictoriaLogs API                                                                                                                                                                                                                           | No       | -                | -                                |
| `VL_INSTANCE_HEADERS`      | Custom HTTP headers to send with requests (comma-separated key=value pairs)                                                                                                                                                                                         | No       | -                | -                                |
| `MCP_PASSTHROUGH_HEADERS`  | HTTP header names to forward from incoming MCP requests to VictoriaLogs (comma-separated list). Overrides `VL_INSTANCE_HEADERS` on collision. Only applies in `sse`/`http` modes.                                                                                   | No       | -                | -                                |
| `VL_DEFAULT_TENANT_ID`     | Default tenant ID used when tenant is not specified in requests (format: `AccountID:ProjectID` or `AccountID`)                                                                                                                                                      | No       | `0:0`            | -                                |
| `MCP_SERVER_MODE`          | Server operation mode. See [Modes](#modes) for details.                                                                                                                                                                                                             | No       | `stdio`          | `stdio`, `sse`, `http`           |
| `MCP_LISTEN_ADDR`          | Address for SSE or HTTP server to listen on                                                                                                                                                                                                                         | No       | `localhost:8081` | -                                |
| `MCP_DISABLED_TOOLS`       | Comma-separated list of tools to disable                                                                                                                                                                                                                            | No       | -                | -                                |
| `MCP_HEARTBEAT_INTERVAL`   | Defines the heartbeat interval for the streamable-http protocol. <br /> It means the MCP server will send a heartbeat to the client through the GET connection, <br /> to keep the connection alive from being closed by the network infrastructure (e.g. gateways) | No       | `30s`            | -                                |
| `MCP_LOG_FORMAT`           | Log output format                                                                                                                                                                                                                                                   | No       | `text`           | `text`, `json`                   |
| `MCP_LOG_LEVEL`            | Minimum log level                                                                                                                                                                                                                                                   | No       | `info`           | `debug`, `info`, `warn`, `error` |

### Modes

MCP Server supports the following modes of operation (transports):

- `stdio` - Standard input/output mode, where the server reads commands from standard input and writes responses to standard output. This is the default mode and is suitable for local servers.
- `sse` - Server-Sent Events. Server will expose the `/sse` and `/message` endpoints for SSE connections.
- `http` - Streamable HTTP. Server will expose the `/mcp` endpoint for HTTP connections.

More info about traqnsports you can find in MCP docs:

- [Core concepts -> Transports](https://modelcontextprotocol.io/docs/concepts/transports)
- [Specifications -> Transports](https://modelcontextprotocol.io/specification/2025-03-26/basic/transports)

### Сonfiguration examples

```bash
# For a public playground
export VL_INSTANCE_ENTRYPOINT="https://play-vmlogs.victoriametrics.com"

# Custom headers for authentication (e.g., behind a reverse proxy)
# Expected syntax is key=value separated by commas
export VL_INSTANCE_HEADERS="<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"

# Forward specific headers from incoming MCP requests to VictoriaLogs
export MCP_PASSTHROUGH_HEADERS="X-Token,X-Access-Key"

# Server mode
export MCP_SERVER_MODE="sse"
export MCP_SSE_ADDR="0.0.0.0:8081"
export MCP_DISABLED_TOOLS="hits,facets"
```

## Endpoints

In SSE and HTTP modes the MCP server provides the following endpoints:

| Endpoint            | Description                                                                                      |
|---------------------|--------------------------------------------------------------------------------------------------|
| `/`                 | Landing page with setup help and tool inspection (HTTP mode only)                                |
| `/sse` + `/message` | Endpoints for messages in SSE mode (for MCP clients that support SSE)                            |
| `/mcp`              | HTTP endpoint for streaming messages in HTTP mode (for MCP clients that support Streamable HTTP) |
| `/metrics`          | Metrics in Prometheus format for monitoring the MCP server                                       |
| `/health/liveness`  | Liveness check endpoint to ensure the server is running                                          |
| `/health/readiness` | Readiness check endpoint to ensure the server is ready to accept requests                        |

## Setup in clients

### Cursor

Go to: `Settings` -> `Cursor Settings` -> `MCP` -> `Add new global MCP server` and paste the following configuration into your Cursor `~/.cursor/mcp.json` file:

```json
{
  "mcpServers": {
    "victorialogs": {
      "command": "/path/to/mcp-victorialogs",
      "env": {
        "VL_INSTANCE_ENTRYPOINT": "<YOUR_VL_INSTANCE>",
        "VL_INSTANCE_BEARER_TOKEN": "<YOUR_VL_BEARER_TOKEN>",
        "VL_INSTANCE_HEADERS": "<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
      }
    }
  }
}
```

See [Cursor MCP docs](https://docs.cursor.com/context/model-context-protocol) for more info.

### Claude Desktop

Add this to your Claude Desktop `claude_desktop_config.json` file (you can find it if open `Settings` -> `Developer` -> `Edit config`):

```json
{
  "mcpServers": {
    "victorialogs": {
      "command": "/path/to/mcp-victorialogs",
      "env": {
        "VL_INSTANCE_ENTRYPOINT": "<YOUR_VL_INSTANCE>",
        "VL_INSTANCE_BEARER_TOKEN": "<YOUR_VL_BEARER_TOKEN>",
        "VL_INSTANCE_HEADERS": "<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
      }
    }
  }
}
```

See [Claude Desktop MCP docs](https://modelcontextprotocol.io/quickstart/user) for more info.

### Claude Code

Run the command:

```sh
claude mcp add victorialogs -- /path/to/mcp-victorialogs \
  -e VL_INSTANCE_ENTRYPOINT=<YOUR_VL_INSTANCE> \
  -e VL_INSTANCE_BEARER_TOKEN=<YOUR_VL_BEARER_TOKEN> \
  -e VL_INSTANCE_HEADERS="<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
```

See [Claude Code MCP docs](https://docs.anthropic.com/en/docs/agents-and-tools/claude-code/tutorials#set-up-model-context-protocol-mcp) for more info.

### Codex

Codex CLI and the IDE extension use the same MCP configuration file: `~/.codex/config.toml`
(or `.codex/config.toml` in a trusted project).

Run the command:

```sh
codex mcp add victorialogs \
  --env VL_INSTANCE_ENTRYPOINT=<YOUR_VL_INSTANCE> \
  --env VL_INSTANCE_BEARER_TOKEN=<YOUR_VL_BEARER_TOKEN> \
  --env VL_INSTANCE_HEADERS="<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>" \
  -- /path/to/mcp-victorialogs
```

Or add the following to your Codex `~/.codex/config.toml` file:

```toml
[mcp_servers.victorialogs]
command = "/path/to/mcp-victorialogs"

[mcp_servers.victorialogs.env]
VL_INSTANCE_ENTRYPOINT = "<YOUR_VL_INSTANCE>"
VL_INSTANCE_BEARER_TOKEN = "<YOUR_VL_BEARER_TOKEN>"
VL_INSTANCE_HEADERS = "<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
```

If you run the server in Streamable HTTP mode, you can register it with:

```sh
codex mcp add victorialogs --url http://localhost:8081/mcp
```

See [Codex MCP docs](https://developers.openai.com/codex/mcp) for more info.

### Visual Studio Code

Add this to your VS Code MCP config file:

```json
{
  "servers": {
    "victorialogs": {
      "type": "stdio",
      "command": "/path/to/mcp-victorialogs",
      "env": {
        "VL_INSTANCE_ENTRYPOINT": "<YOUR_VL_INSTANCE>",
        "VL_INSTANCE_BEARER_TOKEN": "<YOUR_VL_BEARER_TOKEN>",
        "VL_INSTANCE_HEADERS": "<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
      }
    }
  }
}
```

See [VS Code MCP docs](https://code.visualstudio.com/docs/copilot/chat/mcp-servers) for more info.

### Zed

Add the following to your Zed config file:

```json
  "context_servers": {
    "victorialogs": {
      "command": {
        "path": "/path/to/mcp-victorialogs",
        "args": [],
        "env": {
                  "VL_INSTANCE_ENTRYPOINT": "<YOUR_VL_INSTANCE>",
        "VL_INSTANCE_BEARER_TOKEN": "<YOUR_VL_BEARER_TOKEN>",
        "VL_INSTANCE_HEADERS": "<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
      }
      },
      "settings": {}
    }
  }
}
```

See [Zed MCP docs](https://zed.dev/docs/ai/mcp) for more info.

### JetBrains IDEs

- Open `Settings` -> `Tools` -> `AI Assistant` -> `Model Context Protocol (MCP)`.
- Click `Add (+)`
- Select `As JSON`
- Put the following to the input field:

```json
{
  "mcpServers": {
    "victorialogs": {
      "command": "/path/to/mcp-victorialogs",
      "env": {
        "VL_INSTANCE_ENTRYPOINT": "<YOUR_VL_INSTANCE>",
        "VL_INSTANCE_BEARER_TOKEN": "<YOUR_VL_BEARER_TOKEN>",
        "VL_INSTANCE_HEADERS": "<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
      }
    }
  }
}
```

### Windsurf

Add the following to your Windsurf MCP config file.

```json
{
  "mcpServers": {
    "victorialogs": {
      "command": "/path/to/mcp-victorialogs",
      "env": {
        "VL_INSTANCE_ENTRYPOINT": "<YOUR_VL_INSTANCE>",
        "VL_INSTANCE_BEARER_TOKEN": "<YOUR_VL_BEARER_TOKEN>",
        "VL_INSTANCE_HEADERS": "<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
      }
    }
  }
}
```

See [Windsurf MCP docs](https://docs.windsurf.com/windsurf/mcp) for more info.

### Using Docker instead of binary

You can run VictoriaLogs MCP Server using Docker instead of local binary.

You should replace run command in configuration examples above in the following way:

```
{
  "mcpServers": {
    "victorialogs": {
      "command": "docker",
        "args": [
          "run",
          "-i", "--rm",
          "-e", "VL_INSTANCE_ENTRYPOINT",
          "-e", "VL_INSTANCE_BEARER_TOKEN",
          "-e", "VL_INSTANCE_HEADERS",
          "ghcr.io/victoriametrics/mcp-victorialogs",
        ],
      "env": {
        "VL_INSTANCE_ENTRYPOINT": "<YOUR_VL_INSTANCE>",
        "VL_INSTANCE_BEARER_TOKEN": "<YOUR_VL_BEARER_TOKEN>",
        "VL_INSTANCE_HEADERS": "<HEADER>=<HEADER_VALUE>,<HEADER>=<HEADER_VALUE>"
      }
    }
  }
}
```

## Usage

After [installing](#installation) and [configuring](#setup-in-clients) the MCP server, you can start using it with your favorite MCP client.

You can start dialog with AI assistant from the phrase:

```
Use MCP VictoriaLogs in the following answers
```

But it's not required, you can just start asking questions and the assistant will automatically use the tools and documentation to provide you with the best answers.

### Toolset

MCP VictoriaLogs provides numerous tools for interacting with your VictoriaLogs instance.

Here's a list of available tools:

| Tool                 | Description                                           |
|----------------------|-------------------------------------------------------|
| `documentation`      | Search in embedded VictoriaLogs documentation         |
| `facets`             | Most frequent values per each log field               |
| `field_names`        | List of field names for the query                     |
| `field_values`       | List of field values for the query                    |
| `flags`              | View non-default flags of the VictoriaLogs instance   |
| `hits`               | The number of matching log entries grouped by buckets |
| `query`              | Execute LogsQL queries                                |
| `stats_query`        | Querying log stats for the given time                 |
| `stats_query_range`  | Querying log stats on the given time range            |
| `stream_field_names` | List of stream fields for the query                   |
| `stream_field_values` | List of stream field values for the query             |
| `stream_ids`         | List of stream IDs for the query                      |
| `streams`            | List of streams for the query                         |

### Prompts

The server includes pre-defined prompts for common tasks.

These are just examples at the moment, the prompt library will be added to in the future:

| Prompt | Description                                           |
|--------|-------------------------------------------------------|
| `documentation` | Search VictoriaLogs documentation for specific topics |

## FAQ

### Why is the MCP server using more resources than I would expect from a simple API proxy?

The server contains an embedded vector database with VictoriaMetrics documentation and blog posts for the `documentation` tool.
It helps to answer complex questions about VictoriaLogs without providing all data to LLM.  
This is the main source of resource usage. To reduce it, add `documentation` to `MCP_DISABLED_TOOLS` environment variable to completely disable the vector database loading.

### How to use one MCP server instance for several VictoriaMetrics instances?

You can use `MCP_PASSTHROUGH_HEADERS` parameter in the MCP Server together with [Header-based routing in vmauth](https://docs.victoriametrics.com/victoriametrics/vmauth/#routing-by-header) to route MCP calls between instances based on HTTP header values from your MCP client config.

## Roadmap

- [ ] Support "Explain query" tool
- [ ] Support optional integration with [VictoriaMetrics Cloud](https://victoriametrics.com/products/cloud/)  
- [ ] Add some extra knowledge to server in addition to current documentation tool:
  - [x] [VictoriaMetrics blog](https://victoriametrics.com/blog/) posts
  - [ ] Github issues
  - [ ] Public slack chat history
  - [ ] CRD schemas
- [ ] Implement multitenant version of MCP (that will support several deployments)
- [ ] Add flags/configs validation tool
- [x] Enabling/disabling tools via configuration

## Disclaimer

AI services and agents along with MCP servers like this cannot guarantee the accuracy, completeness and reliability of results.
You should double check the results obtained with AI.

The quality of the MCP Server and its responses depends very much on the capabilities of your client and the quality of the model you are using.

## Contributing

Contributions to the MCP VictoriaLogs project are welcome! 

Please feel free to submit issues, feature requests, or pull requests.
