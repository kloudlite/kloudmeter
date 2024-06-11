# KloudMeter

KloudMeter is a simple metering tool designed to read and process events of resources, built using Golang and NATS. This tool aims to provide a straightforward solution for monitoring and recording resource usage based on events.

## Features

- Event-based resource metering
- Built with Golang for high performance and efficiency
- Utilizes NATS for lightweight, scalable messaging
- Simple and easy to integrate into existing systems

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [API Endpoints](#api-endpoints)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Installation

### Prerequisites

- Go (version 1.16 or higher)
- NATS server
- Task (a task runner for Go projects)

### Steps

1. Clone the repository:

    ```bash
    git clone https://github.com/kloudlite/kloudmeter.git
    cd kloudmeter
    ```

2. Run the setup tasks:

    ```bash
    task nats:setup
    ```

## Usage

### Running KloudMeter

1. Start the NATS server:

    ```bash
    task nats:start
    ```

2. Run KloudMeter:

    ```bash
    task
    ```

### Example

Register an event through the REST API:

```bash
curl -X POST http://localhost:8080/api/register-event -H "Content-Type: application/json" -d '{
  "id": "unique_event_id",
  "time": "event_timestamp",
  "eventType": "type_of_event",
  "subject": "subject_of_event",
  "data": {
    "key1": "value1",
    "key2": "value2"
  }
}'
```

## Configuration

KloudMeter can be configured via environment variables or a configuration file.

### Environment Variables

- `NATS_URL`: The URL of the NATS server (default: `nats://localhost:4222`).
- `METER_NATS_STREAM`: The NATS stream name for meters (default: `meters`).
- `HTTP_SERVER_PORT`: The port for the HTTP server (default: `8080`).
- `METER_INTERVAL`: The interval (in seconds) for metering (default: `60`).

## API Endpoints

### Register Event

**Endpoint:** `/api/register-event`  
**Method:** `POST`  
**Description:** Registers a new event for metering.  
**Request Body:**

```json
{
  "id": "unique_event_id",
  "time": "event_timestamp",
  "eventType": "type_of_event",
  "subject": "subject_of_event",
  "data": {
    "key1": "value1",
    "key2": "value2"
  }
}
```

### Create Meter

**Endpoint:** `/api/create-meter`  
**Method:** `POST`  
**Description:** Creates a new meter for aggregating event data.  
**Request Body:**

```json
{
  "id": "unique_meter_id",
  "description": "description_of_meter",
  "eventType": "type_of_event_to_aggregate",
  "aggregation": "aggregation_type",
  "valueProperty": "jsonpath(eg $.path)",
  "groupBy": {
    "groupKey": "jsonpath(eg $.path)"
  }
}
```

### List Meters

**Endpoint:** `/api/meters`  
**Method:** `GET`  
**Description:** Retrieves a list of all meters.

### Get Meter

**Endpoint:** `/api/meters/?key={key}`  
**Method:** `GET`  
**Description:** Retrieves details of a specific meter by ID.

### List Readings

**Endpoint:** `/api/readings`  
**Method:** `GET`  
**Description:** Retrieves a list of all readings.

### Get Reading

**Endpoint:** `/api/readings/?key={key}`  
**Method:** `GET`  
**Description:** Retrieves details of a specific reading by ID.

## Development

### Development Environment

To set up a development environment, follow the installation steps and ensure you have Go, NATS, and Task installed.

### Running Tasks

- **Build the project:**

    ```bash
    task build
    ```

- **Run the project:**

    ```bash
    task run
    ```

- **Set up NATS key-value stores and streams:**

    ```bash
    task nats:setup
    ```

- **Start the NATS server:**

    ```bash
    task nats:start
    ```

- **Default task for development:**

    This task watches for changes in `.go` files and rebuilds and reruns the project automatically:

    ```bash
    task
    ```

## Contributing

We welcome contributions from the community! To contribute to KloudMeter, follow these steps:

1. Fork the repository.
2. Create a new branch with a descriptive name.
3. Make your changes and commit them with clear messages.
4. Push your changes to your fork.
5. Create a pull request.

## License

KloudMeter is licensed under the MIT License. See the [LICENSE](LICENSE) file for more details.

---

For more information, visit the [KloudMeter GitHub repository](https://github.com/kloudlite/kloudmeter). If you encounter any issues or have questions, feel free to open an issue or contact the maintainers.

Happy metering!
