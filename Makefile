.PHONY: tidy build recommend rank e2e

tidy:
	cd /data/xiongle/project/recsys_go && go mod tidy

build: recommend rank

recommend:
	cd /data/xiongle/project/recsys_go && go build -o bin/recommend-api ./services/recommend

rank:
	cd /data/xiongle/project/recsys_go && go build -o bin/rank-api ./services/rank

e2e:
	bash /data/xiongle/project/recsys_go/scripts/e2e.sh
