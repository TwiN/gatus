{
  services.gatus = {
    enable = true;

    settings = {
      web.port = 8080;

      endpoints = [
        {
          name = "website";
          url = "https://twin.sh/health";
          interval = "5m";

          conditions = [
            "[STATUS] == 200"
            "[BODY].status == UP"
            "[RESPONSE_TIME] < 300"
          ];
        }
      ];
    };
  };
}
