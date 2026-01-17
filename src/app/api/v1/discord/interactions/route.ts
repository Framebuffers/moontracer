import { NextResponse } from "next/server";

export async function POST(request: Request) {
    try {
        const contentType = request.headers.get('content-type');

        if (!contentType?.includes('application-json')) {
            return NextResponse.json({
                error: 'Content type must be application/json'
            }, { status: 400 })
        }

        const body = await request.json();

        if (body.type === 1) {
            return NextResponse.json({ type: 1 });
        }

        if (body.type === 4) {
            return NextResponse.json({
                type: 4,
                data: { content: 'Received interaction.' }
            });
        }
    } catch(error) {
        console.log('Error: ', error);
        return NextResponse.json({error: 'Failed to process'}, {status: 500});
    }
}