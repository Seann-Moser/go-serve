package example

//func TestProxy(t *testing.T) {
//	logger, err := zap.NewDevelopment()
//	if err != nil {
//		t.Fatal(err)
//	}
//	s := server.NewServer(context.Background(), "8888", logger)
//	if err := s.AddEndpoints(handlers.HealthCheck); err != nil {
//		log.Fatal(err)
//	}
//	if err := s.SubRouterEndpoints("mnlib.com",
//		&endpoints.Endpoint{
//			URL: &url.URL{
//				Path: "/{path}",
//			},
//			Methods:         []string{http.MethodGet, http.MethodPost},
//			PermissionLevel: 0,
//			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
//				_ = json.NewEncoder(w).Encode(map[string]string{
//					"message": "mnlib endpoint",
//				})
//			},
//			Handler: nil,
//		}); err != nil {
//		log.Fatal(err)
//	}
//	if err := s.SubRouterEndpoints("auth.mnlib.com",
//		&endpoints.Endpoint{
//			URL: &url.URL{
//				Path: "/{path}",
//			},
//			Methods:         []string{http.MethodGet, http.MethodPost},
//			PermissionLevel: 0,
//			HandlerFunc: func(w http.ResponseWriter, r *http.Request) {
//				_ = json.NewEncoder(w).Encode(map[string]string{
//					"message": "auth endpoint",
//				})
//			},
//			Handler: nil,
//		}); err != nil {
//		log.Fatal(err)
//	}
//}
