Kind = "service-router"
Name = "payments"
Routes = [
  {
    Match{
      HTTP {
          PathPrefix = "/"
      }  
    }

    Destination {
      RequestTimeout = "10s"
      NumRetries = 3
      RetryOnConnectFailure = true
      RetryOnStatusCodes = [500,501,503]
    }
  }
]
