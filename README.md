# Go-Programming-Blueprints

### commit message rules
```
【fix】       バグ修正
【hostfix】   クリティカルなバグ修正
【add】       新規（ファイル）機能追加
【update】    機能修正
【change】    仕様変更
【clean】     リファクタリング
【disable】   無効化
【remove】    削除
【upgrade】   バージョンアップ
【revert】    変更取り消し
```

### ファイアウォールの設定
```
sudo ufw allow 許可したいポート
sudo ufw default deny
sudo ufw status
```

### ssh
```
ssh hoge@192.168.1.1 -p xx -i ~/.ssh/id_rsa
```

### scp
```
scp -P 22 hoge.txt hoge@192.168.1.1:/home/hoge 
```

### zerossl nginx
```
cat certificate.crt ca_bundle.crt >> test-certificate.crt
sudo cp test-certificate.crt /etc/ssl/
sudo cp private.key /etc/ssl/
```

### install nginx
```
sudo apt install nginx
```

### nginx reload
```
sudo /etc/init.d/nginx reload
```

### グローバルIP調査方法
```
curl httpbin.org/ip
```

### GOPATH設定方法
```
export GOPATH=任意の場所
```

### nsq起動
```
nsqlookup
```

### nsqd起動（ポートが4160の時）
```
nsqd --lookupd-tcp-address=localhost:4160
```

### wsl2:Ubuntu-20.04 MongoDB 状態確認
```
sudo service mongodb status
```

### wsl2:Ubuntu-20.04 MongoDB 起動
```
sudo service mongodb start
```

### $GOPATH 確認方法
```
go env GOPATH
```

### 自動 package import
```
goimports -w *.go
```

### build
```
go build -o hoge
```

### 実行
```
./hoge
```

### wsl:Ubuntu-20.04 mysqlステータス確認
```
sudo service mysql status
[sudo] password for yuto: ****
```

### wsl:Ubuntu-20.04 mysql起動
```
sudo service mysql start
[sudo] password for yuto: ****
```

### mysqlログイン
```
sudo mysql -u root -p
Enter password: *********************
```

### データベース一覧表示
```
mysql> show databases;
```

### データベース作成
```
mysql> create database go_programming_blueprints;
```

### データベース選択
```
mysql> use go_programming_blueprints;
```

### テーブル一覧表示
```
mysql> show tables;
```

### テーブル作成
```
mysql> create table `sprinkle` (
  `prefix` varchar(100) not null,
  `suffix` varchar(100) not null
) engine=InnoDB default charset=utf8;
```

### テーブル設計表示
```
mysql> desc sprinkle;
or
mysql> describe sprinkle;
```

### テーブル設計変更（追加）
```
mysql> alter table sprinkle add id int not null auto_increment primary key;
```

### テーブル設計変更（順番：最初）
```
mysql> alter table sprinkle modify id int first;
```

### データ追加
```
mysql> insert into sprinkle values (1,'','');
or
mysql> insert into sprinkle (id, prefix, suffix) values (2, '', '');
```

### 件数取得
```
mysql> select count(id) from sprinkle;
```
