build:
	go tool gotailwind -i input.css -o pb_public/app.css
	go build

tidy:
	go tidy