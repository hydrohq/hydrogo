#!/bin/bash
sed -i '/permissions = append(permissions, surface.Permissions...)/c\
			for _, p := range surface.Permissions {\n\
				if authorizer != nil \&\& authorizer(p) {\n\
					permissions = append(permissions, p)\n\
				}\n\
			}' internal/adminstudio/extension_surface.go
