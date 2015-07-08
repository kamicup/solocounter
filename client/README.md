### 使い方 ###

1. jquery-now-counter.js をダウンロードして利用するサイトに配置。

2. 利用したいところに、このようにコードを挿入。

```html
<div id="NowCounter"> - </div>

<script src="http://code.jquery.com/jquery-1.11.3.min.js"></script>
<script src="jquery-now-counter.js"></script>
<script>
    $('#NowCounter').asConcurrentAccessCounter({ path:'/your-host.com/any-area', server:'http://your-api-server/' });
</script>
```
