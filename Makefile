# Makefile for the Inspector Application
#
# This Makefile provides convenient shortcuts for common Docker Compose commands
# to streamline the application development and deployment process.

# Variables
COMPOSE_FILE := docker-compose-dev.yml
DOCKER_COMPOSE := docker compose -f $(COMPOSE_FILE)

# Default target: start the application
.PHONY: up
up:
	@echo "Starting the application..."
	$(DOCKER_COMPOSE) up

# Stop the application and remove containers
.PHONY: down
down:
	@echo "Stopping the application and removing containers..."
	$(DOCKER_COMPOSE) down

# Build or rebuild services
.PHONY: build
build:
	@echo "Building Docker images..."
	$(DOCKER_COMPOSE) build

# Rebuild and start the application
.PHONY: rebuild
rebuild:
	@echo "Rebuilding and starting the application..."
	$(DOCKER_COMPOSE) up --build

# Stop and remove containers, networks, images, and volumes
.PHONY: clean
clean:
	@echo "Cleaning up all resources..."
	$(DOCKER_COMPOSE) down --volumes --remove-orphans

# Restart the application (down with volumes, then up with build)
.PHONY: restart
restart:
	@echo "Restarting the application..."
	$(DOCKER_COMPOSE) down --volumes
	$(DOCKER_COMPOSE) up --build

# Display application logs
.PHONY: logs
logs:
	@echo "Displaying application logs..."
	$(DOCKER_COMPOSE) logs -f

# Show available make commands
.PHONY: help
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@echo "  up         Start the application"
	@echo "  down       Stop the application and remove containers"
	@echo "  build      Build or rebuild services"
	@echo "  rebuild    Rebuild and start the application"
	@echo "  restart    Restart the application with a fresh build"
	@echo "  clean      Remove containers, networks, images, and volumes"
	@echo "  logs       Display application logs"
	@echo "  help       Show this help message"

