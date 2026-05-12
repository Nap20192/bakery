FROM node:22-alpine AS builder

WORKDIR /src

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

FROM nginx:1.27-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/dist /usr/share/nginx/html

EXPOSE 8080

CMD ["sh", "-c", "printf 'window.__BAKERY_CONFIG__ = { apiBaseUrl: \"%s\" };' \"$VITE_API_BASE_URL\" > /usr/share/nginx/html/config.js && nginx -g 'daemon off;'"]
