<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
div.container {
	margin: 8px;
}
textarea {
	height: 20em;
	width: 32em;
	display: block;
}
form {
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
<li>{{ . }}</li>
{{ end }}
</ul>
</p>
{{ end }}

<form method="post" action="obtain">
<textarea required id="log" name="log" placeholder="ログを貼り付けてください"></textarea>
<input type="submit" value="獲得状況を反映">
</form>

</div></body>
</html>
