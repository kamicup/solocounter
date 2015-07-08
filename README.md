# なにこれ #

- ウェブサイトに script で埋め込んで「直近数分間の閲覧人数（アクセス元 IP アドレス数）」を表示するやつ。

### 特徴 ###

- 超・単純自己完結なマイクロサービス。
- Go で実装したので、高速。
    - Nginx がスタティックページを返すのと同じくらいの処理性能。
- データは実行プロセス内の共有メモリ領域に格納する。
    - KVS を使うと、その通信がボトルネックになるし、オーバーヘッドも生じるので。
- 必要に応じて Redis の Pub/Sub 機能を使って分散処理も可能。

### サーバ ###

- 性能参考値
    - 開発機（MacBook Air, 13-inch, Early 2014）にて
        - go 1.3.3 でビルドしたバイナリ
        - ```ab -c 100 -n 10000``` したら **9000 req/sec** 前後。
        - シミュレーションモードで内部的に 1000 サイト × 100 req/sec の処理を発生させ続け、60秒ウィンドウで 6,000,000 件ぐらいのアドレスを保持してる状態で同じ ab をしたら **7500 req/sec** 前後。（前項の結果からありえない設定だが。）
            - つまり HTTP のオーバーヘッドに比べれば内部データ管理の処理性能は充分。
            - メモリは 1GB ~ 1.5 GB ぐらい占有した。
    - **t2.micro** にて
        - go 1.4.2 でビルドしたバイナリ
        - ```ab -c 100 -n 10000``` したら **8500 req/sec** ぐらい。
        - 分散処理モード（Redis 使う場合）では 6000 req/sec ぐらいになる。（単独モードの 70% ぐらいか。）
    - **Heroku** （フリーアカウント）にて
        - ```git push heroku master``` すると、
            - go 1.3.3 でビルドされた。
            - デプロイ先は https://ancient-savannah-2334.herokuapp.com/ になった。
        - ```ab -c 10 -n 1000``` したら **13 req/sec** 前後。
        - ```ab -c 100 -n 10000``` で接続先を https じゃなく http で試したら、だいたい 260 req/sec ぐらい。
- こういうの動かすなら
    - ディスクアクセスは**しない**ので I/O 性能はほとんど気にしなくて良い。
    - マルチコアなマシンなら環境変数に GOMAXPROCS=8 とかしつつ -parallel オプション付きで起動するとより良いかもしれない。


### クライアント ###

表示したいところに
```html
<div id="Counter"> - </div>
```
を置いて
```html
<script src="//code.jquery.com/jquery-1.11.3.min.js"></script>
<script src="jquery-now-counter.js"></script>
<script>
    $('#Counter').asConcurrentAccessCounter({ path:'/your-host.com/any-area' });
</script>
```
みたいに適用すると、適当にポーリングして表示更新される。

[client/README.md](./client/)