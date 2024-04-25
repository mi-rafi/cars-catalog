run_killgrave:
 docker run -it -d --rm -p 3001:3000 -v $PWD/testdata:/home -w /home friendsofgo/killgrave --host 0.0.0.0

swag: 
 swag init -g cmd/cars/main.go

wire:	
 wire ./cmd/cars 