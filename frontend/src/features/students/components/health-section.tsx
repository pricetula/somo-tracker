"use client";

export function HealthSection({
    _studentId,
    _isCompact,
}: {
    _studentId: string;
    _isCompact: boolean;
}) {
    // const { data: healthData, isLoading, isError } = useStudentHealth(studentId);
    return null;
    // if (isLoading) {
    //     return (
    //         <div className="flex items-center justify-center py-12">
    //             <Loader2 className="text-muted-foreground h-6 w-6 animate-spin" />
    //         </div>
    //     );
    // }

    // if (isError) {
    //     return (
    //         <div className="text-muted-foreground flex flex-col items-center gap-2 py-12">
    //             <HeartPulse className="h-8 w-8" />
    //             <p className="font-medium">Failed to load health data</p>
    //         </div>
    //     );
    // }

    // const incidents = healthData?.incidents ?? [];
    // const profile = healthData?.profile;
    // const incidentLimit = isCompact ? 5 : 10;

    // return (
    //     <div className={isCompact ? "space-y-4" : "space-y-6"}>
    //         {/* Health Profile */}
    //         {profile && (
    //             <div className="bg-muted/30 p-4">
    //                 <h3 className="mb-2  font-semibold">Health Profile</h3>
    //                 <div className="space-y-1 ">
    //                     {profile.blood_group && (
    //                         <p>
    //                             <span className="text-muted-foreground">Blood Group:</span>{" "}
    //                             {profile.blood_group}
    //                         </p>
    //                     )}
    //                     {profile.allergies && profile.allergies.length > 0 && (
    //                         <p>
    //                             <span className="text-muted-foreground">Allergies:</span>{" "}
    //                             {profile.allergies.join(", ")}
    //                         </p>
    //                     )}
    //                     {profile.chronic_conditions && profile.chronic_conditions.length > 0 && (
    //                         <p>
    //                             <span className="text-muted-foreground">Chronic Conditions:</span>{" "}
    //                             {profile.chronic_conditions.join(", ")}
    //                         </p>
    //                     )}
    //                     {profile.emergency_instructions && (
    //                         <p>
    //                             <span className="text-muted-foreground">Emergency Notes:</span>{" "}
    //                             {profile.emergency_instructions}
    //                         </p>
    //                     )}
    //                 </div>
    //             </div>
    //         )}

    //         {/* Medical Incidents */}
    //         <div>
    //             <div className="mb-3 flex items-center justify-between">
    //                 <h3 className=" font-semibold">
    //                     Medical Incidents
    //                     {incidents.length > 0 && (
    //                         <span className="text-muted-foreground ml-2 font-normal">
    //                             ({incidents.length})
    //                         </span>
    //                     )}
    //                 </h3>
    //                 <Button variant="outline" size="sm" asChild>
    //                     <Link href={`/health/students/${studentId}`}>
    //                         <ArrowUpRight className="mr-1 h-3 w-3" />
    //                         Full History
    //                     </Link>
    //                 </Button>
    //             </div>

    //             {incidents.length === 0 ? (
    //                 <div className="text-muted-foreground flex flex-col items-center gap-2 py-8">
    //                     <HeartPulse className="h-8 w-8" />
    //                     <p className="font-medium">No medical incidents</p>
    //                 </div>
    //             ) : (
    //                 <div className="space-y-2">
    //                     {incidents.slice(0, incidentLimit).map((incident) => (
    //                         <div key={incident.id} className="bg-muted/30 p-3">
    //                             <div className="flex items-start justify-between">
    //                                 <div className="space-y-1">
    //                                     <p className=" font-medium">{incident.symptoms}</p>
    //                                     <p className="text-muted-foreground ">
    //                                         {new Date(
    //                                             incident.incident_timestamp
    //                                         ).toLocaleDateString("en-US", {
    //                                             month: "short",
    //                                             day: "numeric",
    //                                             year: "numeric",
    //                                             hour: "2-digit",
    //                                             minute: "2-digit",
    //                                         })}
    //                                         {incident.logged_by_name &&
    //                                             ` \u00b7 ${incident.logged_by_name}`}
    //                                     </p>
    //                                     {incident.action_taken && (
    //                                         <p className="text-muted-foreground mt-1 ">
    //                                             Action: {incident.action_taken}
    //                                         </p>
    //                                     )}
    //                                 </div>
    //                             </div>
    //                         </div>
    //                     ))}
    //                 </div>
    //             )}
    //         </div>
    //     </div>
    // );
}
