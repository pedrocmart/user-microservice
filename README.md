# User Management Microservice

This microservice provides an API for managing users, including functionality for creating, updating, updating password, deleting, and listing users with filters and pagination. It integrates with RabbitMQ to send notifications when user-related changes occur, allowing other systems to react to these updates.

## Architecture
The project follows a layered architecture with clear separation of concerns:
```text
┌─────────────────┐
│      API        │ ← Handlers HTTP (Chi Router)
└────────┬────────┘
         │
┌────────▼────────┐        ┌─────────────────┐
│     Service     │━━━━━━━▶│   Notification  │← Async (RabbitMQ)
└────────┬────────┘        └─────────────────┘
         │      ↑
         │      Business Logic                 
┌────────▼────────┐   
│   Repository    │ ← Data Persistence
└────────┬────────┘                      
         │                               
┌────────▼────────┐
│    Database     │ ← PostgreSQL
└─────────────────┘
```


### Key Components

- **Handlers**: Responsible for receiving HTTP requests, validating inputs, and formatting responses.
- **Service**: Contains business logic and orchestrates operations between repositories and external systems.
- **Repository**: Abstracts database access and implements CRUD operations.
- **Notification**: System to notify other services about changes in user entities via RabbitMQ.
- **Configuration**: Manages application settings via environment variables.
- **Logs**: Structured logging system using Zap for high-performance logging.



### Data Model

The user model includes the following fields:

- **ID**: Unique UUID (e.g., "26ef0140-c436-4838-a271-32652c72f6f2")
- **FirstName**: First name (e.g., "John")
- **LastName**: Last name (e.g., "Travolta")
- **Nickname**: Unique nickname (e.g., "John123")
- **Password**: Hashed password (e.g., "password12@")
- **Email**: Unique email address (e.g., "john@gggmail.com")
- **Country**: Country (e.g., "US")
- **CreatedAt**: Creation timestamp (e.g., "2024-07-15T07:25:55.32Z")
- **UpdatedAt**: Last update timestamp (e.g., "2024-07-15T07:25:55.32Z")

### HTTP API Endpoints
The HTTP API is available by default at: http://localhost:8080 .

Instead of detailing every API endpoint, this microservice exposes a Swagger UI for easily exploring and testing all available endpoints. 

You can explore and test all available endpoints using the interactive Swagger UI:

Swagger UI - http://localhost:8080/swagger/index.html

Available endpoints:
- **POST /users** - Create a new user
- **GET /users/{id}** - Get a user by ID
- **PUT /users/{id}** - Update an existing user
- **DELETE /users/{id}** - Remove a user
- **PUT /users/{id}/password** - Update a user's password
- **GET /users** - List users with filters and pagination
- **GET /health** - Check the health of the service
- **GET /readiness** - Check if the service is ready to receive traffic

### Filters and Pagination

The user list supports the following parameters:

- **firstname**: Filter by first name
- **lastname**: Filter by last name
- **country**: Filter by country
- **email**: Filter by email
- **nickname**: Filter by nickname
- **page**: Page number (default: 1)
- **page_size**: Page size (default: 10, max: 100)


### Notification System

The service uses RabbitMQ to notify other systems of changes to user entities. Notifications are published to a RabbitMQ queue in an asynchronous manner to ensure the main flow of execution is not blocked. The following events are triggered:

- **user.created**: Published when a new user is successfully created.
- **user.updated**: Published when a user's information is updated.
- **user.deleted**: Published when a user is deleted from the system.

Consumers of these events can subscribe to the relevant queues to perform actions based on the notifications.

The default URL for the RabbitMQ UI is:

http://localhost:15672/    
user: guest | password: guest

In this UI, you can check the status of the queues, inspect messages, and configure exchanges and bindings.

By default, the flag `RABBITMQ_ENABLE_CONSUMER` is set to `false`, which means that messages will be visible in RabbitMQ but will not be automatically consumed by the service. This allows you to inspect the messages before they are consumed.

If you prefer to see the messages being consumed automatically, you can change the flag to `true` in the `.env` file.

### Technologies Used

- **Go 1.24**: Programming language
- **Chi Router**: Lightweight and fast HTTP framework
- **PostgreSQL**: Relational database
- **sqlx**: Extensions for `database/sql`
- **Zap**: Structured high-performance logging
- **RabbitMQ**: Message broker for notifications
- **Docker & Docker Compose**: Containerization
- **Testify**: Testing framework
- **UUID**: Unique identifier generation
- **Bcrypt**: Password hashing
- **Config**: Configuration management via environment variables

### Configuration

The service is configured using environment variables. Below is an example of how to set up the configuration in a `.env` file:

- **SERVER_PORT**: Server port (default: 8080)
- **DB_HOST**: Database host (default: localhost)
- **DB_PORT**: Database port (default: 5432)
- **DB_USER**: Database user (default: postgres)
- **DB_PASSWORD**: Database password (default: postgres)
- **DB_NAME**: Database name (default: users)
- **DB_SSL_MODE**: SSL mode for database connection (default: disable)
- **LOG_LEVEL**: Log level (default: info)
- **NOTIFICATION_SUBSCRIBERS**: List of URLs for notifications, comma-separated
- **RABBITMQ_URL**: RabbitMQ connection URL
- **RABBITMQ_QUEUE_NAME**: Name of the RabbitMQ queue
- **RABBITMQ_ENABLE_CONSUMER**: Whether to enable the consumer (true/false | default: false)


### Running the Service

#### With Docker Compose

```bash
make docker-run
```

### Running Tests

The project includes a suite of unit and integration tests to ensure the system works as expected.
- **Run all tests**:
```bash
make test
````

- **Run tests with coverage**:
```bash
make test-coverage
```


### Design Decisions
- **Layered Architecture**: Clear separation of concerns for easy testing and maintenance.
- **Dependency Injection**: All dependencies are injected, making it easy to swap implementations and test components.
- **Go Idioms**: The code follows Go idioms for simplicity, readability, and performance.
- **Error Handling**: Errors are wrapped with context using the errors package.
- **Structured Logs**: Logs are handled using zap for high-performance, structured logging.
- **Asynchronous Notifications**: Notifications are sent in goroutines to avoid blocking the main flow.
- **Validation**: Input validation is performed rigorously to ensure data integrity.
- **Health Checks**: Specific endpoints are included for service health monitoring.
- **Containerization**: Docker and Docker Compose are used for easy deployment and development.

#### Future Improvements
- **Authentication and Authorization**: Implement JWT or OAuth2 for user authentication.
- **Caching**: Implement caching to improve performance.
- **Rate Limiting**: Add rate limiting to protect the API from abuse.
- **Metrics**: Collect and expose metrics for monitoring.
- **Tracing**: Implement distributed tracing for better observability.
- **CI/CD**: Set up a continuous integration and delivery pipeline.


#### .env file
Just in case you're not seeing the .env file, here is how it should look like:
```
APP_ENV=development
DB_PASSWORD=postgres
DB_HOST=user-postgres
DB_PORT=5432
DB_USER=postgres
DB_NAME=users
DB_SSL_MODE=disable

RABBITMQ_URL=amqp://guest:guest@user-rabbitmq:5672/
RABBITMQ_QUEUE_NAME=user_notifications
RABBITMQ_ENABLE_CONSUMER=false
```

--------

### The Task
To write a small microservice to manage Users. The service should be implemented in Go.

Requirements
A user must be stored using the following schema:

| Field            | Value                             |
|------------------|-----------------------------------|
| ID               | 26ef0140-c436-4838-a271-32652c72f6f2 |
| First name       | Alice                             |
| Last name        | Bob                               |
| Nickname         | AB123                              |
| Password (hashed)| supersecurepassword               |
| Email            | alice@bob.com                     |
| Country          | UK                                |
| Created at       | 2024-07-15T07:25:55.32Z           |
| Updated at       | 2024-07-15T07:25:55.32Z           |

The service must allow you to:
- Add a new User
- Modify an existing User
- Remove a User
- Return a paginated list of Users, allowing for filtering by certain criteria (e.g. all Users with the country "UK")


The service must:
- Provide an HTTP or gRPC API
- Use a sensible storage mechanism for the Users
- Have the ability to notify other interested services of changes to User entities
- Have meaningful logs
- Be well documented - a good balance between meaningful code documentation and general documentation on your choices in a readme file
- Have a health check
The service must NOT:
- Provide login or authentication functionality

Notes: 
It is up to you what technologies and patterns you use to implement these features, but you
will be assessed on these choices and we expect you to be confident in explaining them.
We encourage the use of local alternatives or stubs (for instance a database containerized
and linked to your service through docker-compose).
Some of the considerations we would like you to think about while implementing your
solution are:
- How have you structured your application, and what implications does this have on
future feature requests?
- How would your solution react in a distributed environment with high throughput
traffic?
- How is your code following Go idioms?
- How would you expand the solution if you would have spent more time on it?

`The feedback was not great 🙃`