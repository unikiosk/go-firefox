
setup-profile:
	mkdir -p ./assets/profile

run-firefix:
	firefox --profile ./assets/profile --start-debugger-server 9222 --kiosk https://synpse.net

clean:
	rm -rf ./assets/profile/*