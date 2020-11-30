cd ./frontend/controlcenter
npm install
rm -r ../../www
vue build --dest ../../www ./src/main.js
