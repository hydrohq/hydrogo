# Builds the storefront. Run: docker build -t my-shop . && docker run -p 3000:3000 --env-file .env my-shop
FROM node:22-alpine AS build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build && npm prune --omit=dev

FROM node:22-alpine
WORKDIR /app
ENV NODE_ENV=production PORT=3000
COPY --from=build /app/build ./build
COPY --from=build /app/package.json ./
EXPOSE 3000
CMD ["node", "build"]
