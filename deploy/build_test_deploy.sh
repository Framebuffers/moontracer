#!/bin/bash
set -e

# Configuration
PROJECT_NAME="moontracer"
TEMP_BUILD_DIR="/tmp/docker-builds/$PROJECT_NAME"
# Fix: Now points to ~/source/repos/moontracer (no double nesting)
SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REMOTE_SERVER="framebuffer@10.69.0.2"
REMOTE_PATH="~/source/repos/moontracer.git"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

usage() {
    echo "Usage: $0 [build|test|deploy|all]"
    echo ""
    echo "Commands:"
    echo "  build   - Build Docker image from native WSL path"
    echo "  test    - Build and run tests locally"
    echo "  deploy  - Push to production server"
    echo "  all     - Build, test, and deploy"
    exit 1
}

build() {
    echo -e "${BLUE}=== Building Docker Image ===${NC}"
    echo "Source directory: $SOURCE_DIR"
    echo "Temp build directory: $TEMP_BUILD_DIR"
    
    # Verify we're in the right place
    if [ ! -f "$SOURCE_DIR/package.json" ]; then
        echo -e "${RED}✗ package.json not found in $SOURCE_DIR${NC}"
        echo "Current directory structure:"
        ls -la "$SOURCE_DIR"
        exit 1
    fi
    
    # Clean previous
    if [ -d "$TEMP_BUILD_DIR" ]; then
        echo -e "${YELLOW}Cleaning previous build...${NC}"
        rm -rf "$TEMP_BUILD_DIR"
    fi
    
    # Copy to native filesystem (INCLUDING secrets)
    echo -e "${YELLOW}Copying to native WSL path...${NC}"
    mkdir -p "$(dirname "$TEMP_BUILD_DIR")"
    rsync -a --exclude='.git' --exclude='node_modules' --exclude='.next' \
          "$SOURCE_DIR/" "$TEMP_BUILD_DIR/"
    
    # Verify secrets were copied
    echo -e "${YELLOW}Verifying secrets directory...${NC}"
    if [ -d "$TEMP_BUILD_DIR/secrets" ]; then
        echo -e "${GREEN}✓ Secrets directory exists${NC}"
        ls -la "$TEMP_BUILD_DIR/secrets/" | grep -E "\.txt$" || echo "  (no .txt files found)"
    else
        echo -e "${RED}✗ Secrets directory NOT copied!${NC}"
        exit 1
    fi
    
    # Build from temp directory
    echo -e "${YELLOW}Building (this is fast from native path)...${NC}"
    cd "$TEMP_BUILD_DIR"
    
    # Verify files copied correctly
    if [ ! -f "docker-compose.yml" ]; then
        echo -e "${RED}✗ docker-compose.yml not found in $TEMP_BUILD_DIR${NC}"
        echo "Files in temp directory:"
        ls -la
        exit 1
    fi
    
    # Time the build
    START_TIME=$(date +%s)
    docker compose build
    END_TIME=$(date +%s)
    
    BUILD_TIME=$((END_TIME - START_TIME))
    echo -e "${GREEN}✓ Build completed in ${BUILD_TIME}s${NC}"
    
    # Tag with timestamp
    TIMESTAMP=$(date +%Y%m%d-%H%M%S)
    docker tag moontracer-api:latest moontracer-api:$TIMESTAMP
    echo -e "${GREEN}✓ Tagged as moontracer-api:$TIMESTAMP${NC}"
    
    # Return to source directory before cleanup
    cd "$SOURCE_DIR"
    
    # Cleanup
    echo -e "${YELLOW}Cleaning up temporary files...${NC}"
    rm -rf "$TEMP_BUILD_DIR"
    
    return 0
}

test_local() {
    echo -e "${BLUE}=== Testing Locally ===${NC}"
    
    # Build first if not already built
    if ! docker image inspect moontracer-api:latest &> /dev/null; then
        echo -e "${YELLOW}Image not found, building first...${NC}"
        build
    fi
    
    # Run from source directory (where docker-compose.yml lives)
    cd "$SOURCE_DIR"
    
    # Start containers
    echo -e "${YELLOW}Starting test containers...${NC}"
    docker compose up -d
    
    # Wait for health check
    echo -e "${YELLOW}Waiting for service to be ready...${NC}"
    sleep 5
    
    # Check if secrets are mounted in container
    echo -e "${YELLOW}Checking secrets in container...${NC}"
    docker exec moontracer-api ls -la /run/secrets/ || echo "Warning: Could not check secrets"
    
    # Test endpoint
    echo -e "${YELLOW}Testing /api/v1/test endpoint...${NC}"
    HTTP_CODE=$(curl -s -o /tmp/response.json -w "%{http_code}" http://localhost:3000/api/v1/test)
    RESPONSE=$(cat /tmp/response.json)
    
    if [ "$HTTP_CODE" = "200" ] && [[ "$RESPONSE" == *"ok"* ]]; then
        echo -e "${GREEN}✓ Health check passed (HTTP $HTTP_CODE)${NC}"
        echo "$RESPONSE" | jq . 2>/dev/null || echo "$RESPONSE"
    else
        echo -e "${RED}✗ Health check failed (HTTP $HTTP_CODE)${NC}"
        echo "Response (first 30 lines):"
        cat /tmp/response.json | head -30
        echo ""
        echo "Container logs:"
        docker compose logs --tail=50 moontracer-api
        rm -f /tmp/response.json
        docker compose down
        exit 1
    fi
    
    rm -f /tmp/response.json
    
    # Show recent logs
    echo -e "${YELLOW}Recent logs:${NC}"
    docker compose logs --tail=20 moontracer-api
    
    # Cleanup
    echo -e "${YELLOW}Stopping test containers...${NC}"
    docker compose down
    
    echo -e "${GREEN}✓ Tests passed${NC}"
}

deploy() {
    echo -e "${BLUE}=== Deploying to Production ===${NC}"
    
    # Return to source directory
    cd "$SOURCE_DIR"
    
    # Check git status
    if [[ -n $(git status -s) ]]; then
        echo -e "${RED}✗ Uncommitted changes detected${NC}"
        echo "Commit or stash your changes first:"
        git status -s
        exit 1
    fi
    
    # Push to GitHub (backup)
    echo -e "${YELLOW}Pushing to GitHub...${NC}"
    git push origin main
    
    # Deploy to production
    echo -e "${YELLOW}Deploying to production server...${NC}"
    git push prod main
    
    # Check deployment status
    echo -e "${YELLOW}Checking deployment status...${NC}"
    sleep 5
    
    ssh $REMOTE_SERVER "docker ps | grep moontracer"
    
    echo -e "${GREEN}✓ Deployed to production${NC}"
    echo -e "${BLUE}Test at: https://moontracer.framebuffer.cl/api/v1/test${NC}"
}

# Debug info
echo -e "${BLUE}=== Path Information ===${NC}"
echo -e "Script: ${BASH_SOURCE[0]}"
echo -e "Source: $SOURCE_DIR"
echo ""

# Main script
case "${1:-build}" in
    build)
        build
        ;;
    test)
        build
        test_local
        ;;
    deploy)
        deploy
        ;;
    all)
        build
        test_local
        deploy
        ;;
    *)
        usage
        ;;
esac
