export const MOBILE_BREAKPOINT_PX = 768;
export const MOBILE_MEDIA_QUERY = `(max-width: ${MOBILE_BREAKPOINT_PX}px)`;

export interface MobileTitleView {
  kind: string;
  accountId?: string;
  perspectiveId?: string;
}

export interface NamedEntity {
  id: string;
  name: string;
}

export function isMobileViewportWidth(width: number): boolean {
  return Number.isFinite(width) && width > 0 && width <= MOBILE_BREAKPOINT_PX;
}

export function getMobileHeaderTitle(
  view: MobileTitleView,
  accounts: readonly NamedEntity[],
  perspectives: readonly NamedEntity[]
): string {
  switch (view.kind) {
    case 'account': {
      const acc = accounts.find((a) => a.id === view.accountId);
      return acc ? acc.name : 'Account';
    }
    case 'perspective': {
      const p = perspectives.find((item) => item.id === view.perspectiveId);
      return p ? p.name : 'Perspective';
    }
    case 'bills':
      return 'Bills';
    case 'incomes':
      return 'Income';
    case 'new-account':
      return 'New Account';
    case 'new-perspective':
      return 'New Perspective';
    case 'user':
      return 'User Details';
    default:
      return 'Dashboard';
  }
}
