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
        className="w-full rounded-xl border-sidebar-border/70 bg-white/55 px-[0.6875rem] shadow-none hover:bg-white/85 focus-visible:ring-2 data-[size=default]:h-12 group-data-[collapsible=icon]:size-10! group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:px-1! group-data-[collapsible=icon]:[&>svg]:hidden"
      >
        <div className="flex min-w-0 items-center gap-2.5">
          <div className="flex size-8 shrink-0 items-center justify-center rounded-[0.65rem] bg-sidebar-primary text-sidebar-primary-foreground shadow-sm">
            <span className="text-[0.68rem] font-bold tracking-wide">
              {currentTeam?.name?.slice(0, 2).toUpperCase() || "TE"}
            </span>
          </div>
          <div className="min-w-0 text-left group-data-[collapsible=icon]:hidden">
            <span className="block truncate text-sm font-semibold leading-4">
              {currentTeam?.name || "Select team"}
            </span>
            <span className="block truncate text-[0.68rem] leading-4 text-muted-foreground">
              {currentTeamSlug || "Workspace"}
            </span>
          </div>
        </div>
      </SelectTrigger>
      <SelectContent className="rounded-xl border-border/80 p-0 shadow-xl">
        <div className="px-2 py-1.5 text-[0.68rem] font-semibold uppercase tracking-[0.14em] text-muted-foreground">
          Your teams
        </div>
        {teams.map((team) => (
          <SelectItem key={team.id} value={team.slug}>
            <div className="flex items-center gap-2">
              <div className="flex size-7 items-center justify-center rounded-lg bg-muted">
                <span className="text-[0.65rem] font-bold tracking-wide">
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
