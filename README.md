# Go-Programming-Blueprints

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
```

### テーブル設計変更（追加）
```
mysql> alter table sprinkle add id int not null auto_increment primary key;
```

### テーブル設計変更（順番：最初）
```
mysql> alter table sprinkle modify id int first;
```
