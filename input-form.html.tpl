<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{ .title }}</title>
<style>
div.container {
	display: flex;
	justify-content: center;
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
<body>
<div class="container">

<form method="post" action="check">
<textarea required id="log" name="log" placeholder="ログを貼り付けてください"></textarea>
<input type="submit" value="分配を確認">
</form>

<form method="post" action="obtain">
<textarea required id="log" name="log" placeholder="ログを貼り付けてください"></textarea>
<input type="submit" value="獲得状況を反映">
</form>

</div>
</body>
</html>
