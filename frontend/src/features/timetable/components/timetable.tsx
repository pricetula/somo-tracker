"use client";

import { TimeSlot } from "./timeslot";

// Fake data matching backend API shapes
const ACADEMIC_YEAR_ID = "ay-2024";

const timeBlocks = [
    {
        id: "b1",
        day_of_week: 1,
        period_name: "Period 1",
        start_time: "08:00",
        end_time: "08:40",
        is_break: false,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b2",
        day_of_week: 1,
        period_name: "Period 2",
        start_time: "08:40",
        end_time: "09:20",
        is_break: false,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b3",
        day_of_week: 1,
        period_name: "Period 3",
        start_time: "09:20",
        end_time: "10:00",
        is_break: false,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b4",
        day_of_week: 1,
        period_name: "Short Break",
        start_time: "10:00",
        end_time: "10:20",
        is_break: true,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b5",
        day_of_week: 1,
        period_name: "Period 4",
        start_time: "10:20",
        end_time: "11:00",
        is_break: false,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b6",
        day_of_week: 1,
        period_name: "Period 5",
        start_time: "11:00",
        end_time: "11:40",
        is_break: false,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b7",
        day_of_week: 1,
        period_name: "Lunch Break",
        start_time: "11:40",
        end_time: "12:30",
        is_break: true,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b8",
        day_of_week: 1,
        period_name: "Period 6",
        start_time: "12:30",
        end_time: "13:10",
        is_break: false,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b9",
        day_of_week: 1,
        period_name: "Period 7",
        start_time: "13:10",
        end_time: "13:50",
        is_break: false,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
    {
        id: "b10",
        day_of_week: 1,
        period_name: "Period 8",
        start_time: "13:50",
        end_time: "14:30",
        is_break: false,
        academic_year_id: ACADEMIC_YEAR_ID,
    },
];

const enrichedSlots = [
    // Monday - Grade 7A
    {
        id: "s1",
        structure_id: "b1",
        class_id: "g7a",
        class_name: "Grade 7A",
        period_name: "Period 1",
        day_of_week: 1,
        start_time: "08:00",
        end_time: "08:40",
        is_break: false,
        learning_area_name: "Mathematics",
        teacher_name: "Mr. Mwangi",
        room_identifier: "Room 1",
    },
    {
        id: "s2",
        structure_id: "b2",
        class_id: "g7a",
        class_name: "Grade 7A",
        period_name: "Period 2",
        day_of_week: 1,
        start_time: "08:40",
        end_time: "09:20",
        is_break: false,
        learning_area_name: "English",
        teacher_name: "Ms. Kamau",
        room_identifier: "Room 1",
    },
    {
        id: "s3",
        structure_id: "b3",
        class_id: "g7a",
        class_name: "Grade 7A",
        period_name: "Period 3",
        day_of_week: 1,
        start_time: "09:20",
        end_time: "10:00",
        is_break: false,
        learning_area_name: "Kiswahili",
        teacher_name: "Mr. Otieno",
        room_identifier: "Room 1",
    },
    {
        id: "s5",
        structure_id: "b5",
        class_id: "g7a",
        class_name: "Grade 7A",
        period_name: "Period 4",
        day_of_week: 1,
        start_time: "10:20",
        end_time: "11:00",
        is_break: false,
        learning_area_name: "Science",
        teacher_name: "Mrs. Wanjiku",
        room_identifier: "Science Lab",
    },
    {
        id: "s6",
        structure_id: "b6",
        class_id: "g7a",
        class_name: "Grade 7A",
        period_name: "Period 5",
        day_of_week: 1,
        start_time: "11:00",
        end_time: "11:40",
        is_break: false,
        learning_area_name: "Social Studies",
        teacher_name: "Mr. Njoroge",
        room_identifier: "Room 1",
    },
    {
        id: "s8",
        structure_id: "b8",
        class_id: "g7a",
        class_name: "Grade 7A",
        period_name: "Period 6",
        day_of_week: 1,
        start_time: "12:30",
        end_time: "13:10",
        is_break: false,
        learning_area_name: "CRE",
        teacher_name: "Ms. Achieng",
        room_identifier: "Room 1",
    },
    {
        id: "s9",
        structure_id: "b9",
        class_id: "g7a",
        class_name: "Grade 7A",
        period_name: "Period 7",
        day_of_week: 1,
        start_time: "13:10",
        end_time: "13:50",
        is_break: false,
        learning_area_name: "Agriculture",
        teacher_name: "Mr. Mwangi",
        room_identifier: "Room 2",
    },
    {
        id: "s10",
        structure_id: "b10",
        class_id: "g7a",
        class_name: "Grade 7A",
        period_name: "Period 8",
        day_of_week: 1,
        start_time: "13:50",
        end_time: "14:30",
        is_break: false,
        learning_area_name: "Pre-Technical",
        teacher_name: "Ms. Kamau",
        room_identifier: "Computer Lab",
    },
];

export function TimeTable() {
    const days = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday"];

    return (
        <article className="relative max-h-150 w-full overflow-auto rounded-md border text-xs">
            <table className="w-full">
                <thead className="border-b tracking-wider">
                    <tr>
                        <th
                            scope="col"
                            className="bg-background sticky top-0 left-0 z-30 min-w-50 border-r pl-4 text-left md:w-34"
                        >
                            Time Slots
                        </th>
                        {days.map((day) => (
                            <th
                                key={day}
                                scope="col"
                                className="bg-background sticky top-0 z-20 min-w-50 border-r py-2 pl-4 text-left"
                            >
                                {day}
                            </th>
                        ))}
                    </tr>
                </thead>
                <tbody className="divide-y">
                    {Array.from({ length: 90 }, (_, i) => i).map((i) => (
                        <tr key={i}>
                            <td
                                scope="row"
                                className="bg-background sticky left-0 z-10 border-r pl-4"
                            >
                                <div>Lesson 1</div>
                                <div>08:00 - 08:40</div>
                            </td>

                            <td className="border-r pl-4">
                                <TimeSlot />
                            </td>

                            <td className="border-r pl-4">
                                <div className="cursor-pointer">
                                    <p>English</p>
                                    <p>Ms. Achieng</p>
                                    <span>Room 102</span>
                                </div>
                            </td>

                            <td className="border-r pl-4">
                                <button>
                                    <span>+ Assign</span>
                                </button>
                            </td>

                            <td className="border-r pl-4">
                                <div className="cursor-pointer">
                                    <p>Mathematics</p>
                                    <p>Mr. Omondi</p>
                                    <span>Room 101</span>
                                </div>
                            </td>

                            <td className="border-r pl-4">
                                <div className="cursor-pointer">
                                    <p>Science & Tech</p>
                                    <p>Ms. Wanjiku</p>
                                    <span>Lab 1</span>
                                </div>
                            </td>
                        </tr>
                    ))}
                </tbody>
            </table>
        </article>
    );
}
