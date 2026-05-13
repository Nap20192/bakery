FROM node:22-alpine AS builder

WORKDIR /src/frontend

COPY package.json package-lock.json ./
RUN npm ci

COPY . ./
RUN npm run build

FROM node:22-alpine

ENV HOST=0.0.0.0
ENV PORT=8080
ENV BACKEND_URL=http://bakery.railway.internal:8080

WORKDIR /app

COPY server.js ./server.js
COPY --from=builder /src/frontend/dist ./dist

EXPOSE 8080

CMD ["node", "server.js"]
