build:
	go tool gotailwind -i input.css -o pb_public/app.css
	go build

air:
	go tool air

tidy:
	go tidy
