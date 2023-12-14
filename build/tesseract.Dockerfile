FROM node:20.8.0

WORKDIR /tmp

RUN npm install tesseract-wasm@0.10.0

COPY /tmp/node_modules/tesseract-wasm/dist/