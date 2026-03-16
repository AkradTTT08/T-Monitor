# Build stage
FROM node:22-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm install

COPY . .

RUN npm run build

# Final stage
FROM node:22-alpine

WORKDIR /app

# Copy necessary files for production
COPY --from=builder /app/package*.json ./
COPY --from=builder /app/build ./build
COPY --from=builder /app/node_modules ./node_modules

EXPOSE 3001

# Run the SvelteKit app using Node.js
CMD ["node", "build/index.js"]
