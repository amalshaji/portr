import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from "@/components/ui/select";
import type { Team } from "@/types";

export default function TeamSelector() {
  const { team: currentTeamSlug } = useParams<{ team: string }>();
  const [teams, setTeams] = useState<Team[]>([]);
  const [currentTeam, setCurrentTeam] = useState<Team | null>(null);

  useEffect(() => {
    const getMyTeams = async () => {
      try {
        const response = await fetch("/api/v1/user/me/teams", {
          headers: {
            "Content-Type": "application/json",
          },
        });
        if (response.ok) {
          const teamsData = await response.json();
          setTeams(teamsData);

          // Find current team
          if (currentTeamSlug) {
            const team = teamsData.find(
              (t: Team) => t.slug === currentTeamSlug
            );
            if (team) {
              setCurrentTeam(team);
            }
          }
        }
      } catch (error) {
        console.error("Failed to fetch teams:", error);
      }
    };

    getMyTeams();
  }, [currentTeamSlug]);

  const switchTeams = (value: string) => {
    window.location.href = `/${value}/overview`;
  };

  return (
    <Select value={currentTeamSlug} onValueChange={switchTeams}>
      <SelectTrigger
        aria-label="Switch team"
        className="w-full rounded-md border-sidebar-border bg-background px-2.5 shadow-none hover:bg-sidebar-accent focus-visible:ring-2 data-[size=default]:h-11 group-data-[collapsible=icon]:size-9! group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-1! group-data-[collapsible=icon]:[&>svg]:hidden"
      >
        <div className="flex min-w-0 items-center gap-2.5">
          <div className="flex size-7 shrink-0 items-center justify-center rounded-sm bg-sidebar-primary text-sidebar-primary-foreground">
            <span className="data text-[0.65rem] font-semibold">
              {currentTeam?.name?.slice(0, 2).toUpperCase() || "TE"}
            </span>
          </div>
          <div className="min-w-0 text-left group-data-[collapsible=icon]:hidden">
            <span className="block truncate text-sm font-medium leading-4">
              {currentTeam?.name || "Select team"}
            </span>
            <span className="data block truncate text-[0.68rem] leading-4 text-muted-foreground">
              {currentTeamSlug || "workspace"}
            </span>
          </div>
        </div>
      </SelectTrigger>
      <SelectContent className="p-0">
        <div className="eyebrow px-2 py-1.5">Your teams</div>
        {teams.map((team) => (
          <SelectItem key={team.id} value={team.slug}>
            <div className="flex items-center gap-2">
              <div className="flex size-6 items-center justify-center rounded-sm bg-muted">
                <span className="data text-[0.62rem] font-semibold">
                  {team.name.slice(0, 2).toUpperCase()}
                </span>
              </div>
              <span>{team.name}</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
