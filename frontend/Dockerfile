# Build stage
FROM node:22-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

# Pass VITE_API_URL during build time if needed by some plugins, 
# although we now use a runtime-friendly config.ts
RUN npm run build

# Final stage
FROM node:22-alpine

WORKDIR /app

# Only copy the necessary files for preview/run
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/.svelte-kit ./.svelte-kit
COPY --from=builder /app/build ./build
COPY --from=builder /app/static ./static
COPY --from=builder /app/node_modules ./node_modules

EXPOSE 3001

# Using preview for now as it's configured in package.json to serve the build
CMD ["npm", "run", "preview", "--", "--host", "0.0.0.0"]
