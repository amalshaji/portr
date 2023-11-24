export const getLoggedInUser = async () => {
  const response = await fetch("/api/users/me");
  return await response.json();
};

export const getConnections = async (type: string = "") => {
  const response = await fetch(`/api/connections?type=${type}`);
  return await response.json();
};
