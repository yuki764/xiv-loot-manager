<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
div.container {
	margin: 8px;
}
</style>
</head>
<body><div class="container">
<p><a href="/">フォームに戻る</a></p>

{{ range $loot, $players := .candidates }}
<p>{{ $loot }}
<ul>
{{ range $players }}
<li>{{ . }}
{{ end }}
</ul>
</p>
{{ end }}
</div></body>
</html>
