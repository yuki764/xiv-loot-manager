<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
div.container {
	margin: 8px;
}
textarea {
	height: 10em;
	width: 32em;
	display: block;
}
</style>
</head>
<body><div class="container">
<p><a href="/">フォームに戻る</a></p>

{{ range .obtain }}
<p> {{ .Player }} <- {{ .Item }}
{{ end }}

<form method="post" action="obtain/submit">
<textarea required id="sql" name="sql" readonly>
{{ range .obtain -}}
INSERT INTO `TABLE_NAME` VALUES (GENERATE_UUID(), CURRENT_TIMESTAMP(), "{{ .Player }}", "{{ .Item }}");
{{ end -}}
</textarea>
<input type="submit" value="確定">
</form>

</div></body>
</html>
