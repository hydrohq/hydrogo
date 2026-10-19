/** Small shared UI state: the mobile sidebar drawer and the search dialog. */
class UI {
  drawerOpen = $state(false);
  searchOpen = $state(false);
}

export const ui = new UI();
