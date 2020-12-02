<!DOCTYPE html>
<html lang="it">
<head>
    <title>Inceneritore</title>
    <meta charset="utf-8">
    <meta content="width=device-width,initial-scale=1"name="viewport">
    <link href="css/bootstrap.min.css"rel="stylesheet">
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
        $query = "SELECT Name, Count(inceneriti.UserID) AS num FROM inceneriti, utenti WHERE utenti.UserID = inceneriti.UserID GROUP BY inceneriti.UserID ORDER BY Count(inceneriti.UserID) DESC";
        $result = mysqli_query($connection, $query);
        if (mysqli_num_rows($result) != 0) {
            while ($row = mysqli_fetch_array($result)) {
                echo "<tr>";
                echo "<td>$row[Name]</td>";
                echo "<td>$row[num]</td>";
                echo "</tr>";
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
        $query = "SELECT utenti.Name AS Name, inceneriti.TimeStamp AS TimeStamp, config.serverName AS serverName FROM inceneriti, config, utenti WHERE config.serverId = inceneriti.serverId AND utenti.UserID = inceneriti.UserID ORDER BY inceneriti.TimeStamp DESC";
        $result = mysqli_query($connection, $query);
        if (mysqli_num_rows($result) != 0) {
            while ($row = mysqli_fetch_array($result)) {
                echo "<tr>";
                echo "<td>$row[Name]</td>";
                echo "<td>$row[TimeStamp]</td>";
                echo "<td>$row[serverName]</td>";
                echo "</tr>";
            }
        } else {
            mysqli_close($connection);
        } ?></tbody>
    </table>
</div>
</body>
</html>
