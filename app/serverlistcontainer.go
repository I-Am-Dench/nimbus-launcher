package app

// func (app *App) ServerNames() []string {
// 	return app.servers.Names()
// }

// func (app *App) GetServer(index int) *resource.Server {
// 	return app.servers.GetIndex(index)
// }

// func (app *App) AddServer(server *resource.Server) error {
// 	if server == nil {
// 		return fmt.Errorf("fatal add server error: server is nil")
// 	}

// 	err := app.servers.Add(server)
// 	if err != nil {
// 		return err
// 	}

// 	app.serverSelector.SetOptions(app.servers.Names())
// 	return nil
// }

// func (app *App) RemoveServer(server *resource.Server) error {
// 	if server == nil {
// 		return fmt.Errorf("fatal remove server error: server is nil")
// 	}

// 	err := app.servers.Remove(server.Id)
// 	if err != nil {
// 		return err
// 	}

// 	app.serverSelector.SetOptions(app.servers.Names())
// 	app.Refresh()
// 	return nil
// }

// func (app *App) SaveServers() error {
// 	app.serverSelector.SetOptions(app.servers.Names())
// 	app.Refresh()
// 	return app.servers.SaveInfos()
// }
