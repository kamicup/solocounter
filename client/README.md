### 使い方 ###

1. jquery-now-counter.js をダウンロードして利用するサイトに配置。

2. 利用したいところに、このようにコードを挿入。

```html
<div id="NowCounter"> - </div>

<script src="http://code.jquery.com/jquery-1.11.3.min.js"></script>
<script src="jquery-now-counter.js"></script>
<script>
    $('#NowCounter').asConcurrentAccessCounter({ path:'/your-host.com/path-to-watch', server:'http://your-api-server/' });
</script>
```

オプションの *server* は、指定しなければ Heroku で動いているデモ用のサーバーになります。
別にそのまま使っていただいてもかまいませんが、レスポンス性能は保障はできませんので悪しからずご了解ください。
