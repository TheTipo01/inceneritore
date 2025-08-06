<!DOCTYPE html>
<html lang="it">
<head>
    <title>Inceneritore</title>
    <meta charset="utf-8">
    <meta content="width=device-width,initial-scale=1" name="viewport">
    <link href="css/bootstrap.min.css" rel="stylesheet">
</head>
<body>
<div class="container">
    <h2>Top Inceneriti</h2>
    <p>Leaderboard delle persone incenerite:</p>
    <table class="table table-hover">
        <thead>
        <tr>
            <th>Nome</th>
            <th>Numero</th>
        </tr>
        </thead>
        <tbody><?php $connection = mysqli_connect("127.0.0.1", "user", "pass", "database");
        $query = "SELECT Name, Count(incinerated.userID) AS num FROM incinerated, users WHERE users.id = incinerated.userID GROUP BY incinerated.userID ORDER BY Count(incinerated.userID) DESC";
        $result = mysqli_query($connection, $query);
        if (mysqli_num_rows($result) != 0) {
            while ($row = mysqli_fetch_array($result)) {
                echo "<tr><td>".htmlspecialchars(strip_tags($row["Name"]))."</td><td>".htmlspecialchars(strip_tags($row["num"]))."</td></tr>";
            }
        } else {
            mysqli_close($connection);
        } ?></tbody>
    </table>
</div>
<div class="container">
    <h2>Inceneriti</h2>
    <p>Lista delle persone incenerite:</p>
    <table class="table table-hover">
        <thead>
        <tr>
            <th>Nome</th>
            <th>Ora e data</th>
            <th>Server</th>
        </tr>
        </thead>
        <tbody><?php $connection = mysqli_connect("127.0.0.1", "user", "pass", "database");
        $query = "SELECT users.name AS Name, incinerated.timestamp AS TimeStamp, servers.name AS serverName FROM incinerated, servers, users WHERE servers.id = incinerated.serverID AND users.id = incinerated.userID ORDER BY incinerated.timestamp DESC";
        $result = mysqli_query($connection, $query);
        if (mysqli_num_rows($result) != 0) {
            while ($row = mysqli_fetch_array($result)) {
                echo "<tr><td>".htmlspecialchars(strip_tags($row["Name"]))."</td><td>".htmlspecialchars(strip_tags($row["TimeStamp"]))."</td><td>".htmlspecialchars(strip_tags($row["serverName"]))."</td></tr>";
            }
        } else {
            mysqli_close($connection);
        } ?></tbody>
    </table>
</div>
</body>
</html>
