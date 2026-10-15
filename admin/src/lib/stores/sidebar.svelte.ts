class SidebarState {
  open = $state(true);
  mobileOpen = $state(false);

  toggle() {
    this.open = !this.open;
  }

  toggleMobile() {
    this.mobileOpen = !this.mobileOpen;
  }

  closeMobile() {
    this.mobileOpen = false;
  }
}

export const sidebar = new SidebarState();
