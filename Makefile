build:
	docker build -t octo-server .
push:
	docker tag octo-server registry.cn-shanghai.aliyuncs.com/wukongim/octo-server:latest-ultimate
	docker push registry.cn-shanghai.aliyuncs.com/wukongim/wukongchatserver:latest-ultimate
deploy:
	docker build -t octo-server .
	docker tag octo-server registry.cn-shanghai.aliyuncs.com/wukongim/octo-server:latest-ultimate
	docker push registry.cn-shanghai.aliyuncs.com/wukongim/octo-server:latest-ultimate
run-dev:
	docker-compose build;docker-compose up -d
stop-dev:
	docker-compose stop
env-test:
	docker-compose -f ./testenv/docker-compose.yaml up -d 