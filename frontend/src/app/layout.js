import './globals.css';

export const metadata = {
    title: 'MaaS — Mail as a Service',
    description: 'Lightweight multi-user, multi-domain email platform',
};

export default function RootLayout({ children }) {
    return (
        <html lang="en">
            <body>{children}</body>
        </html>
    );
}
