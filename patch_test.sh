#!/bin/bash
sed -i 's/other.ID, other.Name = "billing", "Billing"/other.ID, other.Name = "billing", "Billing"\n\tother.Surfaces = \[\]extensionv1.AdminSurface{{ID: "home", Label: "Home", Path: "\/billing", Placement: "billing"}}/' internal/adminstudio/extensions_test.go
